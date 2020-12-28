package controller

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"

	"zhaokun.org/xdp-lb/pkg/lbmap"
)

type (
	// RedirectRule provides methods to operate ebpf map from the HTTP server
	RedirectRule interface {
		Run() <-chan error
	}

	lbRule struct {
		mapper lbmap.RedirectMetaBPFMapper
		router *httprouter.Router
		addr   string
	}
	// ErrorResp is common response object for server
	ErrorResp struct {
		Message string `json:"message"`
	}

	// Server represent a backend server for loadbalance
	Server struct {
		BackendServer
		ForwardBytes    uint64 `json:"forward_bytes"`
		ForwardPackages uint64 `json:"forward_packages"`
	}

	// BackendServer is data structure that a user want specific a backend server for loadbalance
	BackendServer struct {
		Server  string `json:"server"`
		Mac     string `json:"mac"`
		IFindex uint16 `json:"ifindex"`
	}
)

// NewRedirectRule create a RedirectRule object
func NewRedirectRule(mapper lbmap.RedirectMetaBPFMapper,
	addr string) RedirectRule {
	return &lbRule{mapper, httprouter.New(), addr}
}

func (l *lbRule) Run() <-chan error {
	l.router.GET("/rules", adapter(l.getRuleStatistics))
	l.router.POST("/rules", adapter(l.updateRedirectRules))
	c := make(chan error)
	func() {
		c <- http.ListenAndServe(l.addr, l.router)
	}()
	log.Printf("Server started...")
	return c
}

func adapter(f func(*http.Request, httprouter.Params) (interface{}, error)) httprouter.Handle {
	return func(resp http.ResponseWriter, request *http.Request, params httprouter.Params) {
		result, err := f(request, params)
		resp.Header().Set("content-type", "application/json")
		if err != nil {
			log.Printf("%s processing error: %+v", request.URL.RequestURI(), err)
			resp.WriteHeader(500)
			response, _ := json.Marshal(ErrorResp{err.Error()})
			resp.Write(response)
			return
		}

		response, _ := json.Marshal(result)
		resp.Write(response)
	}
}

func (l *lbRule) updateRedirectRules(request *http.Request, params httprouter.Params) (interface{}, error) {
	request.ParseForm()
	sourceAddr := request.Form.Get("sourceAddr")
	if sourceAddr == "" {
		return nil, errors.New("sourceAddr is required")
	}
	decorder := json.NewDecoder(request.Body)
	var servers []BackendServer
	err := decorder.Decode(&servers)
	if err != nil {
		return nil, errors.Wrap(err, "decode request error")
	}

	var mapServers []lbmap.BackendServer
	for _, s := range servers {
		sm := lbmap.BackendServer{
			SourceAddr: sourceAddr,
			DestAddr:   s.Server,
			Mac:        s.Mac,
			Ifindex:    s.IFindex,
		}
		mapServers = append(mapServers, sm)
	}
	err = l.mapper.Set(mapServers)
	if err != nil {
		return nil, errors.Wrap(err, "set backend server for map error")
	}
	return ErrorResp{"ok"}, nil
}

func (l *lbRule) getRuleStatistics(req *http.Request, params httprouter.Params) (interface{}, error) {
	servers, err := l.mapper.Get()
	if err != nil {
		return nil, err
	}
	serversResp := []Server{}
	tmpMap := make(map[string]*Server)
	for _, s := range servers {
		server := Server{
			BackendServer: BackendServer{
				Server:  lbmap.InetNtoa(s.DestAddr),
				IFindex: s.IfIndex,
				Mac:     lbmap.MacString(s.Mac),
			},
			ForwardBytes:    s.Bytes,
			ForwardPackages: s.Packages,
			//
		}

		if tmpMap[server.Server] == nil {
			tmpMap[server.Server] = &server
		} else {
			s := tmpMap[server.Server]
			s.ForwardPackages += server.ForwardPackages
			s.ForwardBytes += server.ForwardBytes
		}
	}
	for _, v := range tmpMap {
		serversResp = append(serversResp, *v)
	}
	return serversResp, nil
}
