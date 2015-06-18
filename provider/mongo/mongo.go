// Copyright 2012 Lightpoke. All rights reserved.
// This source code is subject to the terms and
// conditions defined in the "License.txt" file.

// Package mongo implements mongodb session storage.
package mongo

import (
	"github.com/sinni800/organics"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
)

type provider struct {
	collection *mgo.Collection
}

func (p *provider) Save(key string, whatChanged string, s *organics.Store) error {
	sk := bson.M{"session": key}

	n, err := p.collection.Find(sk).Count()
	if err != nil {
		return err
	}

	if n == 0 {
		m := make(bson.M)
		for _, k := range s.Keys() {
			m[k] = s.Get(k, nil)
		}
		_, err := p.collection.Upsert(sk, bson.M{"$set": m})
		if err != nil {
			return err
		}
	} else {
		if !s.Has(whatChanged) {
			_, err := p.collection.Upsert(sk, bson.M{"$unset": bson.M{whatChanged: ""}})
			if err != nil {
				return err
			}
		} else {
			v := s.Get(whatChanged, nil)
			_, err := p.collection.Upsert(sk, bson.M{"$set": bson.M{whatChanged: v}})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *provider) Load(key string) *organics.Store {
	m := make(bson.M)
	err := p.collection.Find(bson.M{"session": key}).One(&m)
	if err != nil {
		log.Println(err)
		return nil
	}

	s := organics.NewStore()

	for k, v := range m {
		if k != "_id" {
			s.Set(k, v)
		}
	}

	return s
}

func Provider(c *mgo.Collection) organics.SessionProvider {
	p := new(provider)
	p.collection = c
	return p
}
