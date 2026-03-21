package discovery

import (
	"bytes"
	"debug/elf"
	"fmt"
	"sort"
)

// InspectELFSections returns sorted section names from ELF bytes.
func InspectELFSections(elfData []byte) ([]string, error) {
	if len(elfData) == 0 {
		return nil, fmt.Errorf("empty ELF data")
	}
	f, err := elf.NewFile(bytes.NewReader(elfData))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var names []string
	for _, s := range f.Sections {
		if s.Name != "" {
			names = append(names, s.Name)
		}
	}
	sort.Strings(names)
	return names, nil
}
