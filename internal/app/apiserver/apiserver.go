package apiserver

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/EwvwGeN/APILearning/internal/app/data"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"
	"time"
)

type APIServer struct {
	config          *Config
	logger          *logrus.Logger
	router          *mux.Router
	usersController *data.DBController
	cache           *data.Cache
}

func New(config *Config) *APIServer {
	return &APIServer{
		config: config,
		logger: logrus.New(),
		router: mux.NewRouter(),
	}
}

func (server *APIServer) Start() error {
	if err := server.configureLogger(); err != nil {
		return err
	}
	server.configureRouter()
	var err error
	server.usersController, err = data.NewController(server.config.MDBCon, server.config.UserDataDB, server.config.UsersCol)
	if err != nil {
		server.logger.Error(err)
		return err
	}
	server.cache = data.NewCache(server.config.CacheLiveTime*time.Minute, server.config.CleaningInterval*time.Minute)
	server.logger.Info("Starting API server")
	return http.ListenAndServe(server.config.BindAddr, server.router)
}

func (server *APIServer) configureLogger() error {
	level, err := logrus.ParseLevel(server.config.LogLevel)
	if err != nil {
		server.logger.Error(err)
		return err
	}
	server.logger.SetLevel(level)
	return nil
}

func (server *APIServer) configureRouter() {
	server.router.HandleFunc("/Auth-token/username={username}", server.generateToken()).Methods("GET")
	server.router.HandleFunc("/Configs/app={app}", server.getConfig()).Methods("GET")
	server.router.HandleFunc("/Configs/app={app}", server.createConfig()).Methods("POST")
	server.router.HandleFunc("/Configs/app={app}", server.updateConfig()).Methods("PUT")
	server.router.HandleFunc("/Configs/app={app}", server.deleteConfig()).Methods("DELETE")
}

func (server *APIServer) getConfig() http.HandlerFunc {
	return server.checkToken(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		appName := mux.Vars(r)["app"]
		hashUsrn := md5.Sum([]byte(r.Context().Value("username").(string)))
		appConfig, err := server.cache.GetAppConfig(server.config.MDBCon, fmt.Sprintf("%x", hashUsrn), appName)
		if err != nil {
			json.NewEncoder(w).Encode(err)
			return
		}
		var acceptedConfig map[string]interface{}
		json.Unmarshal([]byte(appConfig), &acceptedConfig)
		json.NewEncoder(w).Encode(acceptedConfig)
	})
}

func (server *APIServer) createConfig() http.HandlerFunc {
	return server.checkToken(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		jsConfigBit, _ := json.Marshal(body)
		jsConfigString := string(jsConfigBit)

		appName := mux.Vars(r)["app"]
		hashUsrn := fmt.Sprintf("%x", md5.Sum([]byte(r.Context().Value("username").(string))))

		err := server.cache.AddAppConfig(server.config.MDBCon, hashUsrn, appName, jsConfigString)
		if err == nil {
			json.NewEncoder(w).Encode("successful creating config")
		} else {
			json.NewEncoder(w).Encode(err)
		}

	})
}

func (server *APIServer) updateConfig() http.HandlerFunc {
	return server.checkToken(func(w http.ResponseWriter, r *http.Request) {

		appName := r.Context().Value("app")
		hashUsrn := md5.Sum([]byte(r.Context().Value("username").(string)))

		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		jsConfigBit, _ := json.Marshal(body)
		jsConfigString := string(jsConfigBit)

		err := server.cache.UpdateAppConfig(server.config.MDBCon, fmt.Sprintf("%x", hashUsrn), appName.(string), jsConfigString)
		if err != nil {
			json.NewEncoder(w).Encode(jsConfigString)
		} else {
			json.NewEncoder(w).Encode(err)
		}

	})
}

func (server *APIServer) deleteConfig() http.HandlerFunc {
	return server.checkToken(server.checkConfigExist(func(w http.ResponseWriter, r *http.Request) {

		appName := r.Context().Value("app")
		hashUsrn := md5.Sum([]byte(r.Context().Value("username").(string)))

		err := server.cache.DeleteAppConfig(server.config.MDBCon, fmt.Sprintf("%x", hashUsrn), appName.(string))
		if err != nil {
			json.NewEncoder(w).Encode("successful deleting config")
		} else {
			json.NewEncoder(w).Encode(err)
		}
	}))
}

func (server *APIServer) generateToken() http.HandlerFunc {
	return server.checkUsernameLen(server.checkUniqUser(func(w http.ResponseWriter, r *http.Request) {
		username := r.Context().Value("username")
		tokenContainer := make([]byte, 16)
		rand.Read(tokenContainer)
		token := hex.EncodeToString(tokenContainer)
		err := server.usersController.AddUser(username.(string), server.generateSafeToken(token))
		if err != nil {
			server.logger.Error(err)
		}
		answer := map[string]string{
			"username": username.(string),
			"token":    fmt.Sprintf("%s.%s", username, token),
		}
		json.NewEncoder(w).Encode(answer)
	}))
}

func (server *APIServer) generateSafeToken(token string) string {
	safeKey := server.config.ProtectKey
	safeToken := fmt.Sprintf("%x", md5.Sum([]byte(safeKey+token)))
	return safeToken
}

func (server *APIServer) checkUsernameLen(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		params := mux.Vars(r)
		if 1 < len(params["username"]) && len(params["username"]) > 16 {
			http.Error(w, "username should be between 1 and 16 characters", 400)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), "username", params["username"])))
	}
}

func (server *APIServer) checkUniqUser(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		username := r.Context().Value("username")
		user, err := server.usersController.FindUserByName(username.(string))
		if err == nil {
			http.Error(w, "user already registered", 400)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), "user", user)))
	}
}

func (server *APIServer) checkToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authToken := strings.Split(r.Header.Get("Authorization"), ".")
		username := authToken[0]
		acceptedToken, err := server.usersController.GetUserToken(username)
		if err != nil {
			http.Error(w, "incorrect token", 400)
			return
		}
		token := authToken[1]
		safeToken := server.generateSafeToken(token)
		if safeToken != acceptedToken {
			http.Error(w, "incorrect token", 400)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), "username", username)))
	}
}

func (server *APIServer) checkConfigExist(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appName := mux.Vars(r)["app"]
		hashUsrn := md5.Sum([]byte(r.Context().Value("username").(string)))
		appController, _ := data.NewController(server.config.MDBCon, fmt.Sprintf("%x", hashUsrn), appName)
		_, err := appController.FindConfig()
		var exist bool
		if err == nil {
			exist = true
		} else {
			exist = false
		}
		r = r.WithContext(context.WithValue(r.Context(), "configExist", exist))
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), "app", appName)))
	}
}
