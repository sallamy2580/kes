// Copyright 2022 - MinIO, Inc. All rights reserved.
// Use of this source code is governed by the AGPLv3
// license that can be found in the LICENSE file.

package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/minio/kes"
	"github.com/minio/kes/internal/auth"
)

func serverCreateEnclave(mux *http.ServeMux, config *ServerConfig) API {
	const (
		Method  = http.MethodPost
		APIPath = "/v1/enclave/create/"
		MaxBody = 1 << 20
		Timeout = 15 * time.Second
	)
	type Request struct {
		Admin kes.Identity `json:"admin"`
	}
	handler := func(w http.ResponseWriter, r *http.Request) {
		w = audit(w, r, config.AuditLog.Log())

		if r.Method != Method {
			w.Header().Set("Accept", Method)
			Error(w, errMethodNotAllowed)
			return
		}

		if err := normalizeURL(r.URL, APIPath); err != nil {
			Error(w, err)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, MaxBody)

		err := Sync(config.Vault.Locker(), func() error {
			sysAdmin, err := config.Vault.Admin(r.Context())
			if err != nil {
				return err
			}
			if identity := auth.Identify(r); identity != sysAdmin {
				return kes.ErrNotAllowed
			}
			name := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, APIPath))
			if err = validateName(name); err != nil {
				return err
			}
			var req Request
			if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
				return err
			}
			if err = validateName(req.Admin.String()); err != nil {
				return err
			}
			if req.Admin.IsUnknown() {
				return kes.NewError(http.StatusBadRequest, "identity is unknown")
			}
			if req.Admin == sysAdmin {
				return kes.NewError(http.StatusBadRequest, "admin identity cannot system admin")
			}
			if _, err = config.Vault.CreateEnclave(r.Context(), name, req.Admin); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			Error(w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
	mux.HandleFunc(APIPath, timeout(Timeout, proxy(config.Proxy, config.Metrics.Count(config.Metrics.Latency(handler)))))
	return API{
		Method:  Method,
		Path:    APIPath,
		MaxBody: MaxBody,
		Timeout: Timeout,
	}
}

func serverDeleteEnclave(mux *http.ServeMux, config *ServerConfig) API {
	const (
		Method  = http.MethodDelete
		APIPath = "/v1/enclave/delete/"
		MaxBody = 0
		Timeout = 15 * time.Second
	)
	handler := func(w http.ResponseWriter, r *http.Request) {
		w = audit(w, r, config.AuditLog.Log())

		if r.Method != Method {
			w.Header().Set("Accept", Method)
			Error(w, errMethodNotAllowed)
			return
		}

		if err := normalizeURL(r.URL, APIPath); err != nil {
			Error(w, err)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, MaxBody)

		err := Sync(config.Vault.Locker(), func() error {
			sysAdmin, err := config.Vault.Admin(r.Context())
			if err != nil {
				return err
			}
			if identity := auth.Identify(r); identity != sysAdmin {
				return kes.ErrNotAllowed
			}
			name := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, APIPath))
			if err = validateName(name); err != nil {
				return err
			}
			return config.Vault.DeleteEnclave(r.Context(), name)
		})
		if err != nil {
			Error(w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
	mux.HandleFunc(APIPath, timeout(Timeout, proxy(config.Proxy, config.Metrics.Count(config.Metrics.Latency(handler)))))
	return API{
		Method:  Method,
		Path:    APIPath,
		MaxBody: MaxBody,
		Timeout: Timeout,
	}
}
