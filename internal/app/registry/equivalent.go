package registry

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
)

// EquivRegistries contains a map of registry name to list of registry names.
// If a Docker registry notification contains one of the registry names in a
// list then the corresponding key in the map is used as the registry name
// in its place.
//
// e.g. equiv-registries.json ...
//
// {
//   "my.registry.com": [
//     "my.registry.com:443",
//     "my.registry.com:8443"
//   ],
//   "other.registry.co.uk": [
//     "other.registry.com",
//     "other.registry.org"
//   ]
// }
//
type EquivRegistries struct {
	Equivs map[string][]string
}

// FindEquivalent returns the equivalent registry name, or the same
// name if there is no equivalent.
func (e *EquivRegistries) FindEquivalent(inputName string) string {
	for name, equivs := range e.Equivs {
		for _, equiv := range equivs {
			if equiv == inputName {
				log.Printf("equiv of %s is %s\n", inputName, name)
				return name
			}
		}
	}
	return inputName
}

// CreateEquivRegistries creates an equivalent registries object from
// a JSON config file.
func CreateEquivRegistries(equivRegistriesFile string) (*EquivRegistries, error) {
	equivRegistries := &EquivRegistries{}
	var err error
	if equivRegistriesFile != "" {
		var file *os.File
		file, err = os.Open(equivRegistriesFile)
		if err == nil {
			err = json.NewDecoder(bufio.NewReader(file)).Decode(&equivRegistries.Equivs)
		}
	}
	if err != nil {
		equivRegistries = nil
	}
	return equivRegistries, err
}
