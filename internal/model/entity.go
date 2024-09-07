package model

type Entity interface {
	GetID() string
	Hash() (string, error)
}
