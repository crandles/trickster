/*
 * Copyright 2018 The Trickster Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package ur

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	uropt "github.com/trickstercache/trickster/v2/pkg/backends/alb/mech/ur/options"
	"github.com/trickstercache/trickster/v2/pkg/proxy/authenticator/options"
	"github.com/trickstercache/trickster/v2/pkg/proxy/authenticator/providers/basic"
)

func FuzzUserRouterCredentials(f *testing.F) {
	seeds := []string{
		"",
		"alice:secret",
		":onlypass",
		"onlyuser:",
		":",
		"user:pass:withcolon",
		"user\x00name:p\x00ass",
		"\xff\xfe\xfd:\x00\x01",
		strings.Repeat("a", 4096) + ":" + strings.Repeat("b", 4096),
		"用户:密码",
		"  alice  :  secret  ",
		"alice:secret\r\nX-Evil: 1",
		"alice\n:secret",
		"alice:",
		"\x00\x00\x00\x00",
		"a:b\x00",
	}
	for _, s := range seeds {
		f.Add(s)
		f.Add(base64.StdEncoding.EncodeToString([]byte(s)))
	}
	f.Add("not-base64!!!")
	f.Add("Bearer something")

	auth, err := basic.NewPtr(map[string]any{
		"options": &options.Options{Name: "fuzz"},
	})
	if err != nil {
		f.Fatalf("basic.NewPtr: %v", err)
	}
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := &Handler{
		authenticator: auth,
		options: &uropt.Options{
			DefaultHandler: okHandler,
			Users: uropt.UserMappingOptionsByUser{
				"alice": {ToHandler: okHandler},
				"":      {ToHandler: okHandler},
			},
		},
	}

	f.Fuzz(func(t *testing.T, raw string) {
		r, rerr := http.NewRequest("GET", "http://example.com/", nil)
		if rerr != nil {
			return
		}
		if len(raw)%3 == 0 {
			r.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(raw)))
		} else {
			r.Header.Set("Authorization", raw)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Result() == nil {
			t.Fatal("nil response from ServeHTTP")
		}
	})
}
