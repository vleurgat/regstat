package main

import (
	"flag"

	"github.com/vleurgat/regstat/internal/app/regstat"
)

func main() {
	var port string
	var pgConnStr string
	var dockerConfigFile string
	var equivRegistriesFile string
	flag.StringVar(&port, "port", "3333", "the port number to listen on")
	flag.StringVar(&pgConnStr, "pg-conn-str", "", "the Postgres connect string, e.g. \"host=host port=1234 user=user password=pw ...\"")
	flag.StringVar(&dockerConfigFile, "docker-config", "", "the path to the Docker registry config.json file, used to obtain login credentials")
	flag.StringVar(&equivRegistriesFile, "equiv-registries", "", "the path to the equiv-registries.json file, used to combine equivalent registries")
	flag.Parse()
	regstat.Regstat(port, pgConnStr, dockerConfigFile, equivRegistriesFile)
}
