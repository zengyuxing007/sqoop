package graphql

import (
	"fmt"
	"github.com/vektah/gqlgen/graphql"
	"context"
	"github.com/vektah/gqlgen/handler"
	"github.com/solo-io/gloo/pkg/log"
	"github.com/gorilla/mux"
	"net/http"
	"sync"
)

type Server struct {
	routes *routerSwapper
}

func NewGraphQLServer() *Server {
	return &Server{
		routes: &routerSwapper{
			router: mux.NewRouter(),
		},
	}
}

type Endpoint struct {
	// name of the schema this endpoint serves
	SchemaName       string
	// Where the playground will be served
	RootPath   string
	// Where the query path will be served
	QueryPath       string
	// the executable schema to serve
	ExecSchema graphql.ExecutableSchema
}

func (s *Server) UpdateEndpoints(endpoints ... *Endpoint) {
	m := mux.NewRouter()
	for _, endpoint := range endpoints {
		m.Handle(endpoint.RootPath, handler.Playground(endpoint.SchemaName, endpoint.QueryPath))
		m.Handle(endpoint.QueryPath, handler.GraphQL(endpoint.ExecSchema,
			handler.ResolverMiddleware(func(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
				rc := graphql.GetResolverContext(ctx)
				log.Printf("%v: Entered", endpoint.SchemaName, rc.Object, rc.Field.Name)
				res, err = next(ctx)
				log.Printf("%v: Left", endpoint.SchemaName, rc.Object, rc.Field.Name, "=>", res, err)
				return res, err
			}),
		))
	}
	s.routes.swap(m)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.routes.serveHTTP(w, r)
}

// allows changing the routes being served
type routerSwapper struct {
	mu     sync.Mutex
	router *mux.Router
}

func (rs *routerSwapper) swap(newRouter *mux.Router) {
	rs.mu.Lock()
	rs.router = newRouter
	rs.mu.Unlock()
}

func (rs *routerSwapper) serveHTTP(w http.ResponseWriter, r *http.Request) {
	rs.mu.Lock()
	root := rs.router
	rs.mu.Unlock()
	root.ServeHTTP(w, r)
}