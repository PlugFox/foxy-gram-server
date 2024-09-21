package model

type Entity interface {
	GetID() int64
	Hash() (string, error)
}
