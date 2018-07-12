package models

type SshConfiguration struct {
	UserName string `json:"userName"`
	Password string `json:"password"`
	Server string `json:"server"`
	Port int64 `json:"port"`
}