package session

import (
	"container/list"
	"sync"
	"time"
)

type session struct {
	p            *provider
	sid          string                      // session id唯一标示
	timeAccessed time.Time                   // 最后访问时间
	value        map[interface{}]interface{} // session里面存储的值
}

func (st *session) Set(key, value interface{}) error {
	st.value[key] = value
	st.timeAccessed = time.Now()
	return st.p.SessionUpdate(st.sid)
}

func (st *session) Get(key interface{}) interface{} {
	if v, ok := st.value[key]; ok {
		st.timeAccessed = time.Now()
		_ = st.p.SessionUpdate(st.sid)
		return v
	} else {
		return nil
	}
}

func (st *session) Delete(key interface{}) error {
	delete(st.value, key)
	st.timeAccessed = time.Now()
	return st.p.SessionUpdate(st.sid)
}

func (st *session) SessionID() string {
	return st.sid
}

type provider struct {
	lock     sync.Mutex               // 用来锁
	sessions map[string]*list.Element // 用来存储在内存
	list     *list.List               // 用来做gc
}

func (p *provider) SessionInit(sid string) (Session, error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	v := make(map[interface{}]interface{}, 0)
	newness := &session{sid: sid, timeAccessed: time.Now(), value: v, p: p}
	element := p.list.PushBack(newness)
	p.sessions[sid] = element
	return newness, nil
}

func (p *provider) SessionRead(sid string) (Session, error) {
	if element, ok := p.sessions[sid]; ok {
		return element.Value.(*session), nil
	} else {
		sess, err := p.SessionInit(sid)
		return sess, err
	}
}

func (p *provider) SessionDestroy(sid string) error {
	if element, ok := p.sessions[sid]; ok {
		delete(p.sessions, sid)
		p.list.Remove(element)
		return nil
	}
	return nil
}

func (p *provider) SessionGC(maxlifetime int64) {
	p.lock.Lock()
	defer p.lock.Unlock()

	for {
		element := p.list.Back()
		if element == nil {
			break
		}
		if (element.Value.(*session).timeAccessed.Unix() + maxlifetime) < time.Now().Unix() {
			p.list.Remove(element)
			delete(p.sessions, element.Value.(*session).sid)
		} else {
			break
		}
	}
}

func (p *provider) SessionUpdate(sid string) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if element, ok := p.sessions[sid]; ok {
		p.list.MoveToFront(element)
		return nil
	}
	return nil
}

func init() {
	p := &provider{list: list.New(), sessions: map[string]*list.Element{}}
	Register("memory", p)
}
