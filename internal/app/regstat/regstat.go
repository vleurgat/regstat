package regstat

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/distribution/notifications"
	_ "github.com/lib/pq" // import Postgres driver
		"github.com/vleurgat/regstat/internal/app/docker"
	"github.com/vleurgat/regstat/internal/app/registry"
	"github.com/vleurgat/regstat/internal/app/database/postgres"
)

type server struct {
	httpServer *http.Server
	workflow   Workflow
}

func newServer(port string, pgConnStr string, dockerConfig *configfile.ConfigFile, equivRegistries *registry.EquivRegistries) *server {
	s := server{}
	s.httpServer = &http.Server{Addr: ":" + port, Handler: http.HandlerFunc(s.handle)}
	db := postgres.CreateDatabase(pgConnStr)
	db.CreateSchemaIfNecessary()
	client := registry.CreateClient(dockerConfig)
	eqr := equivRegistries
	s.workflow = WorkflowImpl{db: db, client: client, eqr: eqr}
	return &s
}

func (s *server) listenAndServe() error {
	listener, err := net.Listen("tcp", s.httpServer.Addr)
	if err != nil {
		return err
	}
	log.Println("Server now listening on", s.httpServer.Addr)
	s.httpServer.Serve(listener)
	return nil
}

func (s *server) handle(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("error reading request body", err)
		return
	}
	go s.processRegistryRequest(body)
}

func (s *server) processRegistryRequest(body []byte) error {
	if len(body) == 0 {
		return nil
	}
	//log.Println("request body is", string(body))
	var request notifications.Envelope
	err := json.Unmarshal(body, &request)
	if err != nil {
		log.Println("json unmarshal error", err)
		return err
	}
	for _, event := range request.Events {
		log.Printf("event: %s\n", event.Action)
		switch event.Action {
		case "delete":
			s.workflow.processDelete(&event)
		case "pull":
			s.workflow.processPull(&event)
		case "push":
			s.workflow.processPush(&event)
		default:
			log.Println("unknown event action", event.Action)
		}
	}
	return nil
}

// Regstat is the main entry point to the "registry statistics" server. Calling this
// function will start the server listening on the given port for notifications from
// a Docker registry and persisting details of those notifications to the configured
// Postgres database.
func Regstat(port string, pgConnStr string, dockerConfigFile string, equivRegistriesFile string) {
	log.Println("start regstat")

	dockerConfig, err := docker.CreateConfig(dockerConfigFile)
	if err != nil {
		log.Fatalln("failed to process docker config file", dockerConfigFile)
	}

	equivRegistries, err := registry.CreateEquivRegistries(equivRegistriesFile)
	if err != nil {
		log.Fatalln("failed to process equivalent registries file", equivRegistriesFile)
	}

	server := newServer(port, pgConnStr, dockerConfig, equivRegistries)
	server.listenAndServe()
}
