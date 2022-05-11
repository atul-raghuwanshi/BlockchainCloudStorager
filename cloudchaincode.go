package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type DocumentStoreSmartContract struct {
	contractapi.Contract
}

type User struct {
	UserId  string `json:"UserId"`
	UserKey string `json:"UserKey"`
}

type Document struct {
	DocId        string `json:"DocId"`
	Hash         string `json:"Hash"`
	OwnerId      string `json:"OwnerId"`
	SharingId    string `json:"SharingId"`
	EncKey       string `json:"EncKey"`
	Verification string `json:"Verification"`
}

type Dummy struct {
	DummyId  string `json:"UserId"`
	DummyKey string `json:"UserKey"`
}

type Response struct {
	DocumentId string `json:"DocumentId"`
	Content    string `json:"Content"`
}

// This function register a new user id
func (pc *DocumentStoreSmartContract) Register(ctx contractapi.TransactionContextInterface, UserId string) (*User, error) {

	var user string = "UserId"
	user += UserId

	userJSON, err := ctx.GetStub().GetState(user)

	if err != nil {
		return nil, fmt.Errorf("Failed to read the data from world state", err)
	}

	if userJSON != nil {
		return nil, fmt.Errorf("the UserId %s already exists", UserId)
	}

	//generate key
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	s := make([]rune, 10)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	var userkey string = string(s)

	//generate key
	//var userkey string = "1"

	var registration *User = new(User)

	registration.UserId = UserId
	registration.UserKey = userkey

	registrationJson, err := json.Marshal(registration)

	if err != nil {
		return nil, fmt.Errorf("Failed to Marshal the data from world state", err)
	}

	return registration, ctx.GetStub().PutState(user, registrationJson)
}

////////
var aesbyte = []byte{35, 46, 57, 24, 85, 35, 24, 74, 87, 35, 88, 98, 66, 32, 14, 05}

func Encode(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

func Decode(s string) []byte {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return data
}

func Encrypt(text string, MySecret string) (string, error) {
	block, err := aes.NewCipher([]byte(MySecret))
	if err != nil {
		return "", err
	}
	plainText := []byte(text)
	cfb := cipher.NewCFBEncrypter(block, aesbyte)
	cipherText := make([]byte, len(plainText))
	cfb.XORKeyStream(cipherText, plainText)
	return Encode(cipherText), nil
}

func Decrypt(text, MySecret string) (string, error) {
	block, err := aes.NewCipher([]byte(MySecret))
	if err != nil {
		return "", err
	}
	cipherText := Decode(text)
	cfb := cipher.NewCFBDecrypter(block, aesbyte)
	plainText := make([]byte, len(cipherText))
	cfb.XORKeyStream(plainText, cipherText)
	return string(plainText), nil
}
/////////

// This function uploads the document
func (pc *DocumentStoreSmartContract) UploadDocument(ctx contractapi.TransactionContextInterface, Content string, UserKey string, UserId string, DocHash string, ShareId string) (*Document, error) {

	//verify UserId
	var user string = "UserId"
	user += UserId

	userJSON, err := ctx.GetStub().GetState(user)

	if err != nil {
		return nil, err
	}
	if userJSON == nil {
		return nil, fmt.Errorf("User Data not found")
	}
	var registration *User = new(User)

	err = json.Unmarshal(userJSON, &registration)

	if err != nil {
		return nil, err
	}

	if registration.UserKey != UserKey {
		return nil, fmt.Errorf("the UserId and key does not match")
	}

	//verify doc

	//calculate hash sha1 and compare with provided hash
	h := sha1.New()
	h.Write([]byte(Content))
	hs := h.Sum(nil)

	hashstr := fmt.Sprintf("%x", hs)

	if hashstr != DocHash {
		return nil, fmt.Errorf("the Document and Hash does not match")
	}

	//generate document Id
	var temp_doc_id int
	for temp_doc_id = 1; temp_doc_id < 100; temp_doc_id++ {
		tmp := fmt.Sprintf("%d", temp_doc_id)
		var tmp2 string = "DocId"
		tmp2 += tmp

		docJSON, err := ctx.GetStub().GetState(tmp2)

		if err != nil {
			return nil, err
		}

		if docJSON == nil {
			break
		}
	}
	if temp_doc_id == 100 {
		return nil, fmt.Errorf("No documentId available to give")
	}

	final_doc_id := fmt.Sprintf("%d", temp_doc_id)
	final_doc_key := "DocId" + final_doc_id

	//generate enryption key
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	s := make([]rune, 24)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	var EncrKey = string(s)

	//Encypt

	EncyptedContent,_ := Encrypt(Content,EncrKey);

	//upload on DataBase --TODO
	//----------------------------------------
	var requestBody *Response = new(Response)
	requestBody.DocumentId=final_doc_id
	requestBody.Content=EncyptedContent

	requestJson, err := json.Marshal(requestBody)

	if err != nil {
		fmt.Errorf("Failed to Marshal the data from world state", err)
	}

	response,_ :=http.Post("http://172.30.137.4:5000/upload", "application/json", bytes.NewBuffer(requestJson))
	
	if err != nil {
		panic(err)
	}

	defer response.Body.Close()

	content,_ := ioutil.ReadAll(response.Body)

	var responds *Response = new(Response)

	err = json.Unmarshal(content, &responds)

	if err != nil {
		return nil, err
	}
	//-------------------------------------------

	var doc_info *Document = new(Document)

	doc_info.DocId = final_doc_id
	doc_info.EncKey = EncrKey
	doc_info.OwnerId = UserId
	doc_info.Hash = DocHash
	doc_info.SharingId = ShareId
	doc_info.Verification = "NotVerified"

	docinfoJson, err := json.Marshal(doc_info)

	if err != nil {
		return nil, err
	}

	return doc_info, ctx.GetStub().PutState(final_doc_key, docinfoJson)

}

func (pc *DocumentStoreSmartContract) VerifyUpload(ctx contractapi.TransactionContextInterface, EncrKey string, UserKey string, UserId string, DocHash string, ShareId string, DocumentId string) (*Document, error) {

	//verify UserId
	var user string = "UserId"
	user += UserId

	userJSON, err := ctx.GetStub().GetState(user)

	if err != nil {
		return nil, err
	}

	if userJSON == nil {
		return nil, fmt.Errorf("User Data not found")
	}

	var registration *User = new(User)

	err = json.Unmarshal(userJSON, &registration)

	if err != nil {
		return nil, err
	}

	if registration.UserKey != UserKey {
		return nil, fmt.Errorf("the UserId and key does not match")
	}

	var doc string = "DocId"
	doc += DocumentId

	docJSON, err := ctx.GetStub().GetState(doc)

	if err != nil {
		return nil, err
	}

	if docJSON == nil {
		return nil, fmt.Errorf("Document Data not found")
	}

	var docinfo *Document = new(Document)

	err = json.Unmarshal(docJSON, &docinfo)

	if err != nil {
		return nil, err
	}

	//pull document calculate hash check if it is equal
	var requestBody *Response = new(Response)

	requestBody.DocumentId = DocumentId

	requestJson, err := json.Marshal(requestBody)

	if err != nil {
		fmt.Errorf("Failed to Marshal the data from world state", err)
	}

	response,_ :=http.Post("http://172.30.137.4:5000/verify/retrieve", "application/json", bytes.NewBuffer(requestJson))
	
	
	defer response.Body.Close()

	content,_ := ioutil.ReadAll(response.Body)

	var responds *Response = new(Response)

	err = json.Unmarshal(content, &responds)

	decryptedContent,_ := Decrypt(responds.Content,EncrKey)

	//calculate hash of decrypted content
	h := sha1.New()
	h.Write([]byte(decryptedContent))
	hs := h.Sum(nil)

	hashstr := fmt.Sprintf("%x", hs)

	if hashstr != DocHash {
		return nil, fmt.Errorf("Upload Verification failure as hash does not match")
	}
	//check ledger

	if docinfo.OwnerId != UserId {
		return nil, fmt.Errorf("Upload Verification failure as Owner id and User Id does not match")
	}
	if docinfo.SharingId != ShareId {
		return nil, fmt.Errorf("Upload Verification failure as Share id does not match")
	}
	if docinfo.EncKey != EncrKey {
		return nil, fmt.Errorf("Upload Verification failure as Encryption key does not match")
	}

	//send commit to database
	var requestBodycommit *Response = new(Response)

	requestBodycommit.DocumentId = DocumentId

	requestJsoncommit, err := json.Marshal(requestBodycommit)

	if err != nil {
		fmt.Errorf("Failed to Marshal the data from world state", err)
	}

	responsecommit,_ :=http.Post("http://172.30.137.4:5000/verify/upload", "application/json", bytes.NewBuffer(requestJsoncommit))
	
	defer responsecommit.Body.Close()

	contentcommit,_ := ioutil.ReadAll(responsecommit.Body)
	var respondcommit *Response = new(Response)

	err = json.Unmarshal(contentcommit, &respondcommit)

	//-------------------------------------------

	//put verified

	docinfo.Verification = "Verified"

	docinfoJson, err := json.Marshal(docinfo)

	return docinfo, ctx.GetStub().PutState(doc, docinfoJson)
	//return
}

// This function uploads the document
func (pc *DocumentStoreSmartContract) ReadDocument(ctx contractapi.TransactionContextInterface, UserKey string, UserId string, DocumentId string) (*Response, error) {

	//verify UserId
	var user string = "UserId"
	user += UserId

	userJSON, err := ctx.GetStub().GetState(user)

	if err != nil {
		return nil, err
	}

	if userJSON == nil {
		return nil, fmt.Errorf("User Data not found")
	}

	var registration *User = new(User)

	err = json.Unmarshal(userJSON, &registration)

	if err != nil {
		return nil, err
	}

	if registration.UserKey != UserKey {
		return nil, fmt.Errorf("the UserId and key does not match")
	}

	//Read Transaction and check Sharing Status and verify status
	var doc string = "DocId"
	doc += DocumentId

	docJSON, err := ctx.GetStub().GetState(doc)

	if err != nil {
		return nil, err
	}

	if docJSON == nil {
		return nil, fmt.Errorf("Document Data not found")
	}

	var docinfo *Document = new(Document)

	err = json.Unmarshal(docJSON, &docinfo)

	if err != nil {
		return nil, err
	}

	if docinfo.Verification != "Verified" {
		return nil, fmt.Errorf("Document Not Verified")
	}
	if docinfo.SharingId != UserId {
		return nil, fmt.Errorf("Document is inaccessible to user")
	}

	var requestBody *Response = new(Response)	
	requestBody.DocumentId = DocumentId

	requestJson, err := json.Marshal(requestBody)

	if err != nil {
		fmt.Errorf("Failed to Marshal the data from world state", err)
	}

	response,_ :=http.Post("http://172.30.137.4:5000/read", "application/json", bytes.NewBuffer(requestJson))

	defer response.Body.Close()

	content,_ := ioutil.ReadAll(response.Body)

	var responds *Response = new(Response)

	err = json.Unmarshal(content, &responds)

	if err != nil {
		return nil, err
	}

	decryptedContent,_:=Decrypt(responds.Content,docinfo.EncKey)
	
	responds.Content=decryptedContent
	

	//put dummy on ledger
	var dum string = "Dummy"

	dumJSON, err := ctx.GetStub().GetState(dum)

	if dumJSON == nil {
		var dum_info *Dummy = new(Dummy)
		dum_info.DummyId = "Dummy"
		dum_info.DummyKey = "Dummy1"

		dum_infoJson, err := json.Marshal(dum_info)

		if err != nil {
			return nil, err
		}

		return responds, ctx.GetStub().PutState(dum, dum_infoJson)
	}
	var dum_info *Dummy = new(Dummy)

	err = json.Unmarshal(dumJSON, &dum_info)

	if err != nil {
		return nil, err
	}

	if dum_info.DummyKey == "Dummy1" {
		dum_info.DummyKey = "Dummy2"
		dum_infoJson, err := json.Marshal(dum_info)

		if err != nil {
			return nil, err
		}

		return responds, ctx.GetStub().PutState(dum, dum_infoJson)
	}

	dum_info.DummyKey = "Dummy1"
	dum_infoJson, err := json.Marshal(dum_info)

	if err != nil {
		return nil, err
	}

	return responds, ctx.GetStub().PutState(dum, dum_infoJson)
	//return

}

func main() {
	DocStoreSmartContract := new(DocumentStoreSmartContract)

	cc, err := contractapi.NewChaincode(DocStoreSmartContract)

	if err != nil {
		panic(err.Error())
	}

	if err := cc.Start(); err != nil {
		panic(err.Error())
	}
}
