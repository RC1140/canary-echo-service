package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"

	"github.com/NaySoftware/go-fcm"
	"github.com/boltdb/bolt"

	"golang.org/x/crypto/bcrypt"
)

var db *bolt.DB

// ChirpAdditionalData provides additional information for the chirp
type ChirpAdditionalData struct {
	SrcIp     string
	Useragent string
	Referer   string
	Location  string
}

// Chirp notification struct representing
//the data from the canary  token callback
type Chirp struct {
	Manage_url      string
	Memo            string
	Channel         string
	Time            string
	Additional_data ChirpAdditionalData
}

func authedRequestHandler(w http.ResponseWriter, r *http.Request) {
	//u, p, ok := r.BasicAuth()
	//if !ok {
	//		w.Header().Set("WWW-Authenticate", `Basic realm="MY REALM"`)
	//		w.WriteHeader(401)
	//		w.Write([]byte("401 Unauthorized\n"))
	//	}

}

// User is used to record the device token for future push notifications
// Pass this to the register function with no token to create a new user.
type User struct {
	Username string
	Password string
	Token    string
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	var u User
	err := dec.Decode(&u)
	if err != nil {
		log.Println(err)
	}
	if validUser(db, u) {
		if validUserCredentials(db, u) {
			err := updateUserToken(db, u)
			if err != nil {
				log.Println(err)
				http.Error(w, err.Error(), 400)
			} else {
				log.Println(fmt.Sprintf("%s Updated", u.Username))
			}
		} else {
			log.Println(fmt.Sprintf("Invalid creds"))
			http.Error(w, "Invalid creds", 401)
		}
	} else {
		log.Println(fmt.Sprintf("New user found"))
		if viper.GetBool("AllowMultiUserRegistration") {
			log.Println(fmt.Sprintf("New user being registered"))
			setupUser(db, u)

			err = updateUserToken(db, u)
			if err != nil {
				log.Println(err)
				http.Error(w, err.Error(), 400)
			}
		}
	}

	log.Println("OK")
	fmt.Fprintf(w, "OK")

}

func sendNotification(notificationID, message string) {

	ids := []string{
		notificationID,
	}
	data := map[string]string{
		"msg":          message,
		"sum":          message,
		"click_action": "fcm.ACTION.PING",
	}
	c := fcm.NewFcmClient(viper.GetString("ServerAuthToken"))
	c.NewFcmRegIdsMsg(ids, data)

	status, err := c.Send()

	if err == nil {
		status.PrintResults()
	} else {
		fmt.Println(err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "OK")
	dec := json.NewDecoder(r.Body)
	var c Chirp
	err := dec.Decode(&c)
	if err != nil {
		log.Println(err)
		return
	}
	err = recordChirp(db, c)
	if err != nil {
		log.Println(err)
	} else {
		vars := mux.Vars(r)
		var defaultChirper = viper.GetString("AdminUser")
		token := ""
		if chirper, ok := vars["chirper"]; ok {
			defaultChirper = chirper
			token = getUserToken(db, defaultChirper)
			if token == "" {
				if defaultChirper != chirper {
					defaultChirper = viper.GetString("AdminUser")
				}

			}
		}
		token = getUserToken(db, defaultChirper)

		var message = fmt.Sprintf("Canary Token Triggered : %s", c.Memo)
		sendNotification(token, message)
	}
	log.Println(c)
}

func updateUserToken(db *bolt.DB, u User) error {
	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Tokens"))
		if b == nil {
			b, _ = tx.CreateBucket([]byte("Tokens"))
		}
		err := b.Put([]byte(fmt.Sprintf("user-%s", strings.ToLower(u.Username))), []byte(u.Token))
		return err
	})
}

func recordChirp(db *bolt.DB, c Chirp) error {
	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Chirps"))
		if b == nil {
			b, _ = tx.CreateBucket([]byte("Chirps"))
		}
		t := strconv.FormatInt(time.Now().Unix(), 10)
		var test bytes.Buffer
		enc := json.NewEncoder(&test)
		enc.Encode(&c)
		err := b.Put([]byte(t), []byte(test.String()))
		return err
	})
}

func validUser(db *bolt.DB, user User) bool {
	userExists := false
	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Auth"))
		if b == nil {
			return nil
		}
		v := b.Get([]byte(fmt.Sprintf("user-%s", strings.ToLower(user.Username))))
		userExists = (v != nil)
		return nil
	})
	return userExists
}

func getUserToken(db *bolt.DB, username string) string {
	userToken := ""
	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Tokens"))
		if b == nil {
			return nil
		}
		v := b.Get([]byte(fmt.Sprintf("user-%s", strings.ToLower(username))))
		if v != nil {
			userToken = string(v)
			return nil
		}
		return errors.New("Token not found")
	})
	return userToken
}

func validUserCredentials(db *bolt.DB, user User) bool {
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Auth"))
		if b == nil {
			return nil
		}
		v := b.Get([]byte(fmt.Sprintf("user-%s", strings.ToLower(user.Username))))
		if v != nil {
			err := bcrypt.CompareHashAndPassword(v, []byte(user.Password))
			if err != nil {
				return errors.New("Invalid pass")
			}
		}
		return nil
	})
	return err == nil
}

func setupDB() *bolt.DB {
	log.Println("Opening DB")
	db, err := bolt.Open("echo.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	return db
}

func setupUser(db *bolt.DB, u User) error {
	return db.Update(func(tx *bolt.Tx) error {
		var b = tx.Bucket([]byte("Auth"))
		if b == nil {
			b, _ = tx.CreateBucket([]byte("Auth"))
		}
		bytePass, err := bcrypt.GenerateFromPassword([]byte(u.Password), 14)
		if err != nil {
			return err
		}
		err = b.Put([]byte(fmt.Sprintf("user-%s", strings.ToLower(u.Username))), bytePass)
		return err
	})

}

func main() {
	//Default settings to use if a config file is not provided.
	viper.SetDefault("AllowMultiUserRegistration", false)
	viper.SetDefault("WebPort", "8011")
	viper.SetDefault("AdminUser", "Admin")
	viper.SetDefault("AdminPass", "AdminPass")
	viper.SetDefault("ServerAuthToken", "")

	viper.SetConfigType("yaml")
	viper.SetConfigName("echo-config")
	viper.AddConfigPath(".")

	viper.ReadInConfig()

	db = setupDB()
	defer db.Close()

	u := User{Username: viper.GetString("AdminUser"),
		Password: viper.GetString("AdminPass")}
	if !validUser(db, u) {
		setupUser(db, u)
	}

	log.Println("Starting server")
	r := mux.NewRouter()
	r.HandleFunc("/", handler)
	r.HandleFunc("/register-token/", registerHandler)
	r.HandleFunc("/personal/{chirper}/", handler)
	http.Handle("/", r)
	err := http.ListenAndServe(fmt.Sprintf(":%s", viper.Get("WebPort")), nil)
	if err != nil {
		fmt.Println(err.Error())
	}
}
