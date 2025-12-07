package model

type Target struct {
	UID   string   `json:"uid,omitempty"`
	Name  string   `json:"name,omitempty"`
	Note  string   `json:"note"`
	IP    string   `json:"ip,omitempty"`
	CIDR  int      `json:"cidr,omitempty"`
	DType []string `json:"dgraph.type,omitempty"`
}
