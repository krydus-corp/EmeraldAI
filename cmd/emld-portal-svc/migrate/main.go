/*
 * File: main.go
 * Project: migrate
 * File Created: Tuesday, 5th January 2021 8:17:02 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package main

import (
	"crypto/sha1"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"

	db "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/db/mongo"
	models "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/models"
	platform "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/platform"
	secure "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/portal/secure"
)

const (
	defaultUsername  = "admin"
	defaultPassword  = "emerald2022kIk59vEhxsNAY5Xk"
	defaultEmail     = "it@krydus.com"
	defaultDBAddress = "mongodb://root:emld_mongo01@mongo:27017"
)

var (
	username  = flag.String("user", getEnv("MONGO_ADMIN_USERNAME", defaultUsername), "admin username")
	password  = flag.String("password", getEnv("MONGO_ADMIN_PASSWORD", defaultPassword), "admin password")
	email     = flag.String("email", getEnv("MONGO_ADMIN_EMAIL", defaultEmail), "admin email")
	dbAddress = flag.String("database", getEnv("MONGO_URI", defaultDBAddress), "database address")
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func main() {
	flag.Parse()

	db, err := db.New(*dbAddress, 30)
	if err != nil {
		log.Fatalf("Error initializing DB; err=%s", err.Error())
	}
	defer db.Shutdown()

	sec := secure.New(8, sha1.New())

	user := models.User{
		ID:        primitive.NewObjectID(),
		FirstName: *username,
		LastName:  *username,
		Username:  *username,
		Password:  sec.Hash(*password),
		Email:     fmt.Sprintf(*email),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		APIKey:    uuid.New().String(),
		Billing:   models.NewBilling(),
	}

	// User collection initialization
	plat := platform.NewPlatform()
	_, err = plat.UserDB.Create(db, user)
	if err != nil {
		if errors.Is(err, platform.ErrUserAlreadyExists) {
			log.Println("Admin user already exists")
		} else {
			log.Fatalf("Error inserting admin user; err=%s", err.Error())
		}
	} else {
		j, _ := json.MarshalIndent(user, "", "  ")
		log.Printf("Inserted admin user:\n%s\n", string(j))
	}

	// Create indexes
	for _, fn := range plat.Indices() {
		if err := fn(db); err != nil {
			log.Fatalf("Error inserting DB indices; err=%s", err.Error())
		}
	}

	os.Exit(0)
}
