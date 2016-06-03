package http

import (
        "github.com/domeos/agent/funcs"
        "net/http"
)

func configContainerRoutes() {
        http.HandleFunc("/containers", func(w http.ResponseWriter, r *http.Request) {
                RenderDataJson(w, funcs.ContainerStatsForPage())
        })
}

