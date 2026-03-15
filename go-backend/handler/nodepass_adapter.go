package handler

import (
	"bytes"
	"encoding/json"
	"hash/fnv"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type npMachine struct {
	MachineId string `json:"machineId"`
	Status    string `json:"status"`
	Hostname  string `json:"hostname"`
	Ip        string `json:"ip"`
}

type npMapping struct {
	InPort  int    `json:"inPort"`
	OutAddr string `json:"outAddr"`
	OutPort int    `json:"outPort"`
}

type npTunnel struct {
	ID               string      `json:"id"`
	Name             string      `json:"name"`
	Protocol         string      `json:"protocol"`
	IngressMachineId string      `json:"ingressMachineId"`
	EgressMachineId  string      `json:"egressMachineId"`
	Mappings         []npMapping `json:"mappings"`
	Enabled          bool        `json:"enabled"`
}

type npResponse[T any] struct {
	Data T `json:"data"`
}

func useNodePassMode() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("BACKEND_ENGINE")))
	return v == "nodepass"
}

func nodePassControlBase() string {
	v := strings.TrimRight(strings.TrimSpace(os.Getenv("NODEPASS_CONTROL_BASE")), "/")
	if v == "" {
		v = "http://127.0.0.1:3000/api/control"
	}
	return v
}

func stableID(s string) int64 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return int64(h.Sum32())
}

func npReq(method, path string, in any, out any) error {
	var body io.Reader
	if in != nil {
		b, _ := json.Marshal(in)
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, nodePassControlBase()+path, body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	cli := &http.Client{Timeout: 15 * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func npGetMachines() ([]npMachine, error) {
	var res npResponse[[]npMachine]
	err := npReq(http.MethodGet, "/machines", nil, &res)
	return res.Data, err
}

func npGetTunnels() ([]npTunnel, error) {
	var res npResponse[[]npTunnel]
	err := npReq(http.MethodGet, "/tunnels", nil, &res)
	return res.Data, err
}

func npFindMachineIDByNumeric(id int64) (string, bool) {
	machines, err := npGetMachines()
	if err != nil {
		return "", false
	}
	for _, m := range machines {
		if stableID(m.MachineId) == id {
			return m.MachineId, true
		}
	}
	return "", false
}

func npFindTunnelIDByNumeric(id int64) (string, bool) {
	tunnels, err := npGetTunnels()
	if err != nil {
		return "", false
	}
	for _, t := range tunnels {
		if stableID(t.ID) == id {
			return t.ID, true
		}
	}
	return "", false
}
