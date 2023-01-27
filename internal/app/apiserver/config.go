package apiserver

import "time"

type Config struct {
	BindAddr         string        `toml:"bind_addr"`
	LogLevel         string        `toml:"log_level"`
	MDBCon           string        `toml:"mdb_con"`
	UserDataDB       string        `toml:"users_data_db"`
	UsersCol         string        `toml:"users_col"`
	ProtectKey       string        `toml:"protect_key"`
	CacheLiveTime    time.Duration `toml:"cache_live_time_min"`
	CleaningInterval time.Duration `toml:"cleaning_interval_min"`
}

func NewConfig() *Config {
	return &Config{
		BindAddr:         "8080",
		LogLevel:         "debug",
		MDBCon:           "mongodb://localhost:27017",
		UserDataDB:       "usersdata",
		UsersCol:         "users",
		ProtectKey:       "alcohol",
		CacheLiveTime:    10,
		CleaningInterval: 5,
	}
}
