package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"

	"golang.org/x/oauth2"
)

type TokenStore struct {
	fileName string
}

func NewTokenStore(fileName string) *TokenStore {

	return &TokenStore{
		fileName: fileName,
	}
}

func (s *TokenStore) Save(token *oauth2.Token) {

	b, err := json.Marshal(token)
	if err != nil {
		log.Fatal(err)
	}
	ioutil.WriteFile(s.fileName, b, os.ModePerm)
}

func (s *TokenStore) Get() *oauth2.Token {

	var token oauth2.Token
	f, err := os.Open(s.fileName)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		log.Println(err)
		return nil
	}
	err = json.Unmarshal(b, &token)
	if err != nil {
		log.Println(err)
		return nil
	}
	return &token
}
