package model

type Target struct {
	UID   string   `json:"uid,omitempty"`
	Name  string   `json:"target.name,omitempty"`
	Note  string   `json:"target.note"`
	IP    string   `json:"target.ip,omitempty"`
	CIDR  int      `json:"target.cidr,omitempty"`
	DType []string `json:"dgraph.type,omitempty"`
}
