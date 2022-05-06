package toolbox

import (
	"sync"
	"teamide/pkg/form"
	"time"
)

type Worker struct {
	Name       string                                                                                                                `json:"name,omitempty"`
	Text       string                                                                                                                `json:"text,omitempty"`
	Icon       string                                                                                                                `json:"icon,omitempty"`
	Comment    string                                                                                                                `json:"comment,omitempty"`
	Work       func(work string, config map[string]interface{}, data map[string]interface{}) (res map[string]interface{}, err error) `json:"-"`
	ConfigForm *form.Form                                                                                                            `json:"configForm,omitempty"`
}

var (
	workers      *[]*Worker         = &[]*Worker{}
	serviceCache map[string]Service = map[string]Service{}
	lock         sync.Mutex
)

func init() {
	go startServiceTimer()
}

func AddWorker(worker *Worker) {
	*workers = append(*workers, worker)
}

func GetWorkers() (res []*Worker) {
	res = *workers
	return
}

func GetWorker(name string) (res *Worker) {
	for _, one := range *workers {
		if one.Name == name {
			res = one
		}
	}
	return
}

func GetService(key string, create func() (Service, error)) (Service, error) {
	lock.Lock()
	defer lock.Unlock()
	res, ok := serviceCache[key]
	if ok {
		return res, nil
	}
	res, err := create()
	if err != nil {
		return nil, err
	}
	serviceCache[key] = res
	return res, err
}

type Service interface {
	GetWaitTime() int64
	GetLastUseTime() int64
	Stop()
}

func startServiceTimer() {
	for {
		time.Sleep(1 * time.Second)
		nowTime := GetNowTime()
		for key, one := range serviceCache {
			if one.GetWaitTime() <= 0 {
				continue
			}
			t := nowTime - one.GetLastUseTime()
			if t >= one.GetWaitTime() {
				delete(serviceCache, key)
				one.Stop()
			}
		}
	}
}
