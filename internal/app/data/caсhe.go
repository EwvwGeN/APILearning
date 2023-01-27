package data

import (
	"errors"
	"sync"
	"time"
)

type Cache struct {
	sync.RWMutex
	lifeTime         time.Duration
	cleaningInterval time.Duration
	userControllers  map[string]map[string]*mutexDB
}

type mutexDB struct {
	sync.Mutex
	expiration int64
	controller *DBController
	config     string
}

func NewCache(lifeTime, cleaningInterval time.Duration) *Cache {
	cache := Cache{
		lifeTime:         lifeTime,
		cleaningInterval: cleaningInterval,
	}

	go cache.garbageCollector()
	return &cache
}

func (cache *Cache) garbageCollector() {
	for {
		<-time.After(cache.cleaningInterval)
		if cache.userControllers == nil {
			return
		}
		if keys := cache.checkExpired(); len(keys) != 0 {
			cache.clearExpired(keys)

		}
	}
}
func (cache *Cache) checkExpired() (keys map[string][]string) {
	cache.RLock()
	for ukey, ucontroller := range cache.userControllers {
		for akey, dbcontroller := range ucontroller {
			if time.Now().UnixNano() > dbcontroller.expiration && dbcontroller.expiration > 0 {
				keys[ukey] = append(keys[ukey], akey)
			}
		}
	}
	cache.RUnlock()
	return
}

func (cache *Cache) clearExpired(keys map[string][]string) {
	cache.Lock()
	for ukey, appkeys := range keys {
		for akey, _ := range appkeys {
			delete(cache.userControllers[ukey], appkeys[akey])
		}
		if len(cache.userControllers[ukey]) == 0 {
			delete(cache.userControllers, ukey)
		}
	}
	cache.Unlock()
}

func (cache *Cache) addToCache(controller *DBController, username, appname, configbody string) {
	cache.userControllers = make(map[string]map[string]*mutexDB)
	cache.userControllers[username] = make(map[string]*mutexDB)
	cache.userControllers[username][appname] = &mutexDB{
		expiration: time.Now().Add(cache.lifeTime).UnixNano(),
		controller: controller,
		config:     configbody,
	}
}

func (cache *Cache) AddAppConfig(MDBCon, username, appname, configbody string) error {
	var err error
	if cache.userControllers[username][appname] != nil {
		err = errors.New("config already exist")
		return err
	}
	cache.RLock()
	defer cache.RUnlock()
	appDBCOntroller, _ := NewController(MDBCon, username, appname)
	applicationconfig, err := appDBCOntroller.FindConfig()
	if err == nil {
		cache.addToCache(appDBCOntroller, username, appname, applicationconfig)
		err = errors.New("config already exist")
		return err
	}
	appDBCOntroller.AddConfig(configbody)
	cache.addToCache(appDBCOntroller, username, appname, applicationconfig)

	return nil
}

func (cache *Cache) GetAppConfig(MDBCon, username, appname string) (string, error) {
	if cache.userControllers[username][appname] != nil {
		cache.userControllers[username][appname].Lock()
		applicationconfig := cache.userControllers[username][appname].config
		cache.userControllers[username][appname].expiration = time.Now().Add(cache.lifeTime).UnixNano()
		cache.userControllers[username][appname].Unlock()
		return applicationconfig, nil
	} else {
		cache.Lock()
		defer cache.Unlock()
		appDBCOntroller, _ := NewController(MDBCon, username, appname)
		applicationconfig, err := appDBCOntroller.FindConfig()
		if err != nil {
			err := errors.New("config doesnt exist")
			return "", err

		} else {
			cache.addToCache(appDBCOntroller, username, appname, applicationconfig)
			return applicationconfig, nil
		}
	}
}

func (cache *Cache) UpdateAppConfig(MDBCon, username, appname, configbody string) error {
	if cache.userControllers[username][appname] != nil {
		cache.userControllers[username][appname].Lock()
		cache.userControllers[username][appname].controller.UpdateConfig(configbody)
		cache.userControllers[username][appname].config = configbody
		cache.userControllers[username][appname].expiration = time.Now().Add(cache.lifeTime).UnixNano()
		cache.userControllers[username][appname].Unlock()
		return nil
	} else {
		cache.Lock()
		defer cache.Unlock()
		appDBCOntroller, _ := NewController(MDBCon, username, appname)
		err := appDBCOntroller.UpdateConfig(configbody)
		cache.addToCache(appDBCOntroller, username, appname, configbody)
		if err == nil {
			return nil
		} else {
			return errors.New("error while updating config")
		}
	}
}

func (cache *Cache) DeleteAppConfig(MDBCon, username, appname string) error {
	cache.Lock()
	defer cache.Unlock()
	if cache.userControllers[username][appname] != nil {
		cache.userControllers[username][appname].controller.DeleteConfig()
		delete(cache.userControllers[username], appname)
		if len(cache.userControllers[username]) == 0 {
			delete(cache.userControllers, username)
		}
	}
	appDBCOntroller, _ := NewController(MDBCon, username, appname)
	err := appDBCOntroller.DeleteConfig()
	if err == nil {
		return nil
	} else {
		return errors.New("error while deleting config")
	}
}
