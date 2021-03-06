/*
 *  Copyright 2014-2015 LDLN
 *
 *  This file is part of LDLN Base Station.
 *
 *  LDLN Base Station is free software: you can redistribute it and/or modify
 *  it under the terms of the GNU General Public License as published by
 *  the Free Software Foundation, either version 3 of the License, or
 *  any later version.
 *
 *  LDLN Base Station is distributed in the hope that it will be useful,
 *  but WITHOUT ANY WARRANTY; without even the implied warranty of
 *  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *  GNU General Public License for more details.
 *
 *  You should have received a copy of the GNU General Public License
 *  along with LDLN Base Station.  If not, see <http://www.gnu.org/licenses/>.
 */
package controllers

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"github.com/revel/revel"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"strings"
	"github.com/nu7hatch/gouuid"
	"github.com/ldln/core/cryptoWrapper"
)

const salt = "Yp2iD6PcTwB6upati0bPw314GrFWhUy90BIvbJTj5ETbbE8CoViDDGsJS6YHMOBq4VlwW3V00GWUMbbV"

type Web struct {
	*revel.Controller
}

func checkIfSetupIsEligible() bool {

	// connect to mongodb
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// find any deployments
	dbd := session.DB("landline").C("Deployments")
	var resultd map[string]string
	err = dbd.Find(bson.M{}).One(&resultd)

	// find any users
	dbu := session.DB("landline").C("Users")
	var resultu map[string]string
	err = dbu.Find(bson.M{}).One(&resultu)

	// cannot do setup if users exist
	if err != nil {
		return true
	} else {
		return false
	}
}

func (c Web) FirstTimeSetupForm() revel.Result {

	if !checkIfSetupIsEligible() {
		c.Flash.Error("Basestation is already setup")
		return c.Redirect(Web.LoginForm)
	}

	return c.Render()
}

func (c Web) FirstTimeSetupAction(org_title, org_subtitle, org_mbtiles_file, org_map_center_lat, org_map_center_lon, org_map_zoom_min, org_map_zoom_max, org_enc_is_on, username, password, confirm_password string) revel.Result {

	if !checkIfSetupIsEligible() {
		c.Flash.Error("Basestation is already setup")
		return c.Redirect(Web.LoginForm)
	}

	// create deployment
	if createDeployment(org_title, org_subtitle, org_mbtiles_file, org_map_center_lat, org_map_center_lon, org_map_zoom_min, org_map_zoom_max, org_enc_is_on) {

		// create new key for organization
		skek := cryptoWrapper.RandString(32)

		// create first user account
		if createUser(username, password, skek) {
			c.Flash.Success("Organization and user created")
		} else {
			c.Flash.Error("Error generating user")
		}

	} else {
		c.Flash.Error("Error creating organization")
	}

	return c.Redirect(Web.LoginForm)
}

func createDeployment(org_title, org_subtitle, org_mbtiles_file, org_map_center_lat, org_map_center_lon, org_map_zoom_min, org_map_zoom_max, org_enc_is_on string) bool {

	// connect to mongodb
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// save deployment object
	dbu := session.DB("landline").C("Deployments")

	// create object
	deployment_object_map := make(map[string]string)
	uuid, err := uuid.NewV4()
	deployment_object_map["uuid"] = uuid.String()
	deployment_object_map["name"] = org_title
	deployment_object_map["unit"] = org_subtitle
	deployment_object_map["map_center_lat"] = org_map_center_lat
	deployment_object_map["map_center_lon"] = org_map_center_lon
	deployment_object_map["map_zoom_min"] = org_map_zoom_min
	deployment_object_map["map_zoom_max"] = org_map_zoom_max
	deployment_object_map["map_mbtiles"] = org_mbtiles_file
	deployment_object_map["enc_is_on"] = org_enc_is_on

	err = dbu.Insert(deployment_object_map)
	if err != nil {
		panic(err)
	}

	return true
}

func (c Web) requireAuth() bool {
	if c.Session["username"] == "" || c.Session["kek"] == "" {
		revel.AppLog.Debug("Debug: ", "User not authd")
		return false
	}
	revel.AppLog.Debug("Debug: ", "User authd")
	return true
}

func (c Web) WebSocketTest() revel.Result {
	return c.Render()
}

func (c Web) Logout() revel.Result {
	c.Session["username"] = ""
	c.Session["kek"] = ""
	c.Flash.Success("You have logged out successfully")
	return c.Redirect(Web.LoginForm)
}

func (c Web) LoginForm() revel.Result {

	if checkIfSetupIsEligible() {
		return c.Redirect(Web.FirstTimeSetupForm)
	}

	return c.Render()
}

func (c Web) LoginAction(username, password string) revel.Result {

	// hashed_password
	hashed_password := cryptoWrapper.HashPassword(username, password)

	// connect to mongodb
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// find user object
	dbu := session.DB("landline").C("Users")
	var result map[string]string
	err = dbu.Find(bson.M{"username": username, "hashed_password": hashed_password}).One(&result)

	if err != nil {
		revel.AppLog.Debug("Debug: ", "Username and password not found")
	} else {

		// decrypt kek
		ps := []string{password, username, salt}
		key := []byte(string([]rune(strings.Join(ps, "-"))[0:32]))
		bdec, err := hex.DecodeString(result["encrypted_kek"])
		if err != nil {
			revel.AppLog.Debug("Debug: ", err)
			return c.Redirect(Web.LoginForm)
		}
		kek := string(cryptoWrapper.Decrypt(key, bdec))

		// decrypt rsa private
		privenc, err := hex.DecodeString(result["encrypted_rsa_private"])
		if err != nil {
			revel.AppLog.Debug("Debug: ", err)
			return c.Redirect(Web.LoginForm)
		}
		priva := cryptoWrapper.Decrypt(key, privenc)
		priv, err := x509.ParsePKCS1PrivateKey(priva)

		revel.AppLog.Debug("Debug: ", "Login successful")
		revel.AppLog.Debug("Debug: ", username)
		revel.AppLog.Debug("Debug: ", kek)
		revel.AppLog.Debug("Debug: ", priv)

		// get deployment
		dbd := session.DB("landline").C("Deployments")
		var resultd map[string]string
		err = dbd.Find(bson.M{}).One(&resultd)

		// save to session
		c.Session["kek"] = kek
		c.Session["username"] = username
		c.Session["deployment_name"] = resultd["name"]
		c.Session["deployment_unit"] = resultd["unit"]

		// redirect
		return c.Redirect(SyncableObjects.Map)
	}

	// redirect
	c.Flash.Error("Username and password not found")
	return c.Redirect(Web.LoginForm)
}

func (c Web) CreateUserForm() revel.Result {
	if !c.requireAuth() {
		return c.Redirect(Web.LoginForm)
	}
	return c.Render()
}

func (c Web) CreateUserAction(username, password, confirm_password string) revel.Result {

	if !c.requireAuth() {
		return c.Redirect(Web.LoginForm)
	}

	// get kek
	var skek string
	if c.Session["kek"] == "" {
		c.Flash.Error("Error generating user")
		return c.Redirect(Web.CreateUserForm)
	} else {
		skek, ok := c.Session["kek"].(string)
		if ok != true {
			revel.AppLog.Debug("Error: ", "Type conversion")
		}
		revel.AppLog.Debug("Error: ", skek)
	}

	// create user
	if createUser(username, password, skek) {
		c.Flash.Success("User created")
	} else {
		c.Flash.Error("Error generating user")
	}

	// redirect
	return c.Redirect(Web.CreateUserForm)
}

func createUser(username, password, skek string) bool {

	// hashed_password
	hashed_password := cryptoWrapper.HashPassword(username, password)

	// encrypt kek
	ps := []string{password, username, salt}
	key := []byte(string([]rune(strings.Join(ps, "-"))[0:32]))
	pkek := []byte(skek)
	encrypted_kek := hex.EncodeToString(cryptoWrapper.Encrypt(key, pkek))

	// generate rsa keypair for user
	size := 1024
	priv, err := rsa.GenerateKey(rand.Reader, size)
	if err != nil {
		revel.AppLog.Debug("Debug: ", "failed to generate key")
	}
	if bits := priv.N.BitLen(); bits != size {
		revel.AppLog.Debug("Debug: ", "key too short (%d vs %d)", bits, size)
	}
	pub, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	rsa_public_string := hex.EncodeToString(pub)

	revel.AppLog.Debug("Debug: ", priv)

	// encrypt rsa private keypair
	encrypted_rsa_private := hex.EncodeToString(cryptoWrapper.Encrypt(key, x509.MarshalPKCS1PrivateKey(priv)))

	// connect to mongodb
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// save user object
	dbu := session.DB("landline").C("Users")

	user_object_map := make(map[string]string)
	user_object_map["username"] = username
	user_object_map["hashed_password"] = hashed_password
	user_object_map["encrypted_kek"] = encrypted_kek
	user_object_map["encrypted_rsa_private"] = encrypted_rsa_private
	user_object_map["rsa_public"] = rsa_public_string

	err = dbu.Insert(user_object_map)
	if err != nil {
		panic(err)
	}

	return true
}