package actions

import (
	"net/http"

	"github.com/lbryio/lighthouse/app/actions/search"

	"github.com/lbryio/lbry.go/v2/extras/api"
	"github.com/lbryio/lbry.go/v2/extras/orderedmap"
)

type Routes struct {
	m *orderedmap.Map
}

func (r *Routes) Set(key string, h api.Handler) {
	if r.m == nil {
		r.m = orderedmap.New()
	}
	r.m.Set(key, h)
}

func (r *Routes) Each(f func(string, http.Handler)) {
	if r.m == nil {
		return
	}
	for _, k := range r.m.Keys() {
		a, _ := r.m.Get(k)
		f(k, a.(http.Handler))
	}
}

func (r *Routes) Walk(f func(string, http.Handler) http.Handler) {
	if r.m == nil {
		return
	}
	for _, k := range r.m.Keys() {
		a, _ := r.m.Get(k)
		r.m.Set(k, f(k, a.(http.Handler)))
	}
}

func GetRoutes() *Routes {
	routes := Routes{}
	routes.Set("/search", search.Search)
	routes.Set("/autocomplete", AutoComplete)
	routes.Set("/status", Status)

	return &routes
}
