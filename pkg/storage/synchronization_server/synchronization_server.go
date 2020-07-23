package synchronization_server

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/google/uuid"

	"github.com/werf/werf/pkg/util"

	"github.com/werf/lockgate/pkg/distributed_locker"

	"github.com/werf/logboek"
	"github.com/werf/werf/pkg/storage"
)

func RunSynchronizationServer(ip, port string, distributedLockerBackendFactoryFunc func(clientID string) (distributed_locker.DistributedLockerBackend, error), stagesStorageCacheFactoryFunc func(clientID string) (storage.StagesStorageCache, error)) error {
	handler := NewSynchronizationServerHandler(distributedLockerBackendFactoryFunc, stagesStorageCacheFactoryFunc)
	return http.ListenAndServe(fmt.Sprintf("%s:%s", ip, port), handler)
}

type SynchronizationServerHandler struct {
	*http.ServeMux

	DistributedLockerBackendFactoryFunc func(clientID string) (distributed_locker.DistributedLockerBackend, error)
	StagesStorageCacheFactoryFunc       func(clientID string) (storage.StagesStorageCache, error)

	mux                             sync.Mutex
	SyncrhonizationServerByClientID map[string]*SynchronizationServerHandlerByClientID
}

func NewSynchronizationServerHandler(distributedLockerBackendFactoryFunc func(clientID string) (distributed_locker.DistributedLockerBackend, error), stagesStorageCacheFactoryFunc func(requestID string) (storage.StagesStorageCache, error)) *SynchronizationServerHandler {
	srv := &SynchronizationServerHandler{
		ServeMux:                            http.NewServeMux(),
		DistributedLockerBackendFactoryFunc: distributedLockerBackendFactoryFunc,
		StagesStorageCacheFactoryFunc:       stagesStorageCacheFactoryFunc,
		SyncrhonizationServerByClientID:     make(map[string]*SynchronizationServerHandlerByClientID),
	}

	srv.HandleFunc("/new-client-id", srv.handleNewClientID)
	srv.HandleFunc("/", srv.handleRequestByClientID)

	return srv
}

type NewClientIDRequest struct{}
type NewClientIDResponse struct {
	Err      util.SerializableError `json:"err"`
	ClientID string                 `json:"clientID"`
}

func (server *SynchronizationServerHandler) handleNewClientID(w http.ResponseWriter, r *http.Request) {
	var request NewClientIDRequest
	var response NewClientIDResponse
	HandleRequest(w, r, &request, &response, func() {
		logboek.Debug.LogF("SynchronizationServerHandler -- NewClientID request %#v\n", request)
		response.ClientID = uuid.New().String()
		logboek.Debug.LogF("SynchronizationServerHandler -- NewClientID response %#v\n", response)
	})
}

func (server *SynchronizationServerHandler) handleRequestByClientID(w http.ResponseWriter, r *http.Request) {
	logboek.Debug.LogF("SynchronizationServerHandler -- ServeHTTP url path = %q\n", r.URL.Path)

	clientID := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/"), "/", 2)[0]
	logboek.Debug.LogF("SynchronizationServerHandler -- ServeHTTP clientID = %q\n", clientID)

	if clientID == "" {
		http.Error(w, fmt.Sprintf("Bad request: cannot get clientID from URL path %q", r.URL.Path), http.StatusBadRequest)
		return
	}

	if clientServer, err := server.getOrCreateHandlerByClientID(clientID); err != nil {
		http.Error(w, fmt.Sprintf("Internal error: %s", err), http.StatusInternalServerError)
		return
	} else {
		http.StripPrefix(fmt.Sprintf("/%s", clientID), clientServer).ServeHTTP(w, r)
	}
}

func (server *SynchronizationServerHandler) getOrCreateHandlerByClientID(clientID string) (*SynchronizationServerHandlerByClientID, error) {
	server.mux.Lock()
	defer server.mux.Unlock()

	if handler, hasKey := server.SyncrhonizationServerByClientID[clientID]; hasKey {
		return handler, nil
	} else {
		distributedLockerBackend, err := server.DistributedLockerBackendFactoryFunc(clientID)
		if err != nil {
			return nil, fmt.Errorf("unable to create distributed locker backend for clientID %q: %s", clientID, err)
		}

		stagesStorageCache, err := server.StagesStorageCacheFactoryFunc(clientID)
		if err != nil {
			return nil, fmt.Errorf("unable to create stages storage cache for clientID %q: %s", clientID, err)
		}

		handler := NewSynchronizationServerHandlerByClientID(clientID, distributedLockerBackend, stagesStorageCache)
		server.SyncrhonizationServerByClientID[clientID] = handler

		logboek.Debug.LogF("SynchronizationServerHandler -- Created new synchronization server handler by clientID %q: %v\n", clientID, handler)
		return handler, nil
	}
}

type SynchronizationServerHandlerByClientID struct {
	*http.ServeMux
	ClientID string

	DistributedLockerBackend distributed_locker.DistributedLockerBackend
	StagesStorageCache       storage.StagesStorageCache
}

func NewSynchronizationServerHandlerByClientID(clientID string, distributedLockerBackend distributed_locker.DistributedLockerBackend, stagesStorageCache storage.StagesStorageCache) *SynchronizationServerHandlerByClientID {
	srv := &SynchronizationServerHandlerByClientID{
		ServeMux:                 http.NewServeMux(),
		ClientID:                 clientID,
		DistributedLockerBackend: distributedLockerBackend,
		StagesStorageCache:       stagesStorageCache,
	}

	srv.Handle("/locker/", http.StripPrefix("/locker", distributed_locker.NewHttpBackendHandler(srv.DistributedLockerBackend)))
	srv.Handle("/stages-storage-cache/", http.StripPrefix("/stages-storage-cache", NewStagesStorageCacheHttpHandler(stagesStorageCache)))

	return srv
}
