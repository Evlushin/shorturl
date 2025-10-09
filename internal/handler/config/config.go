package config

import "github.com/Evlushin/shorturl/internal/models"

type Config struct {
	ServerAddr string
	BaseAddr   string
	SecretKey  string
	User       models.Users
}
