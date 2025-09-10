package escape

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	gndot "gonum.org/v1/gonum/graph/encoding/dot"
)

type fixture struct {
	name     string   // fixture name
	pkgs     []string // fixture pkgs, like "main", "demo/fixtures/easyPkg/utils"
	statRoot string   // stat root dir of fixture
}

func (f fixture) allcsv() string {
	return filepath.Join(f.statRoot, f.name, "all.csv")
}

func (f fixture) graph(pkg string, batch int, phase int) string {
	return filepath.Join(f.statRoot, f.name, "graphs", pkg, strconv.Itoa(batch), phase2name[phase]+".gv")
}

func (f fixture) csv(pkg string, batch int) string {
	return filepath.Join(f.statRoot, f.name, "csv", pkg, strconv.Itoa(batch), "all.csv")
}

func goroot() string {
	return os.Getenv("GOROOT")
}

func demoDir() string {
	return filepath.Join(goroot(), "demo")
}

func demoStatRootDir() string {
	return filepath.Join(demoDir(), "stat_root")
}

func unmarshalEsg(graphPath string) (*escapeSubGraph, error) {
	bs, err := os.ReadFile(graphPath)
	if err != nil {
		return nil, err
	}
	esg := &escapeSubGraph{}
	err = gndot.Unmarshal(bs, esg)
	if err != nil {
		return nil, err
	}
	return esg, nil
}

func TestGoInterprocedural(t *testing.T) {
	fixture := fixture{
		name: "go_interprocedural",
		pkgs: []string{
			"main",
		},
		statRoot: demoStatRootDir(),
	}
	_, err := unmarshalEsg(fixture.graph("main", 0, 3))
	if err != nil {
		t.Fatal(err)
	}
}
