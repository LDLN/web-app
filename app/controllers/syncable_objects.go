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
	"encoding/json"
	"encoding/hex"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"github.com/revel/revel"
	"github.com/nu7hatch/gouuid"
	"github.com/ldln/web-app/app/routes"
	"github.com/ldln/core/cryptoWrapper"
)

type SyncableObjects struct {
	*revel.Controller
}

func (c SyncableObjects) Map() revel.Result {

	// connect to mongodb
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// find the deployment
	dbd := session.DB("landline").C("Deployments")
	var deployment map[string]string
	err = dbd.Find(bson.M{}).One(&deployment)
	
	// set dek var
	dek := c.Session["kek"]
	
	return c.Render(deployment, dek)
}

func (c SyncableObjects) ListDataTypes() revel.Result {
	
	// connect to mongodb
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()
	
	// query
	dbc := session.DB("landline").C("Schemas")
	var results []map[string]interface{}
	err = dbc.Find(bson.M{}).All(&results)
	if err != nil {
		revel.AppLog.Debug("Debug: ", err)
	}
	
	return c.Render(results)
}

func (c SyncableObjects) CreateDataTypeForm() revel.Result {
	return c.Render()
}

func (c SyncableObjects) CreateDataTypeAction() revel.Result {
	return c.Render()
}

func (c SyncableObjects) CreateObjectForm(object_key string) revel.Result {
	
	// connect to mongodb
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()
	
	// query
	dbc := session.DB("landline").C("Schemas")
	var result map[string]interface{}
	err = dbc.Find(bson.M{"object_key" : object_key}).One(&result)
	if err != nil {
		revel.AppLog.Debug("Debug: ", err)
	}
	
	// chrome
	var hide_chrome bool
	c.Params.Bind(&hide_chrome, "hide_chrome")

	return c.Render(result, hide_chrome)
}

func (c SyncableObjects) CreateObjectAction(object_key string) revel.Result {
	
	revel.AppLog.Debug("Debug: ", c.Params.Values)

	// connect to mongodb
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// find the deployment
	dbd := session.DB("landline").C("Deployments")
	var deployment map[string]string
	err = dbd.Find(bson.M{}).One(&deployment)
	
	// build kv map
	key_values := make(map[string]interface{})
	
	// parse params and put into map
	for k, v := range c.Params.Values {
		if k != "object_key" {
			key_values[k] = v[0]
		}
	}
	revel.AppLog.Debug("Debug: ", key_values)
	
	// convert map to json to string of json
	key_values_map, err := json.Marshal(key_values)
	if err != nil {
		revel.AppLog.Debug("Debug: ", err)
	}
	key_values_string := string(key_values_map[:])
	revel.AppLog.Debug("Debug: ", key_values_string)
	
	// create object
	object_map := make(map[string]interface{})
	uuid, err := uuid.NewV4()
	object_map["uuid"] = uuid.String()
	object_map["object_type"] = object_key
	object_map["time_modified_since_creation"] = float64(0)

	if deployment["enc_is_on"] == "True" {
		// encrypt json string
		kek_bytearr, ok := c.Session["kek"].([]byte)
		if ok != true {
			revel.AppLog.Debug("Error: ", "Type conversion")
		}
		kv_string_encrypted := hex.EncodeToString(cryptoWrapper.Encrypt(kek_bytearr, []byte(key_values_string)))
		revel.AppLog.Debug("Debug: ", kv_string_encrypted)
		
		// test decrypt
		kv_hex, err := hex.DecodeString(kv_string_encrypted)
		if err != nil {
			revel.AppLog.Debug("Debug: ", err)
		}
		kv_plain := string(cryptoWrapper.Decrypt(kek_bytearr, kv_hex))
		revel.AppLog.Debug("Debug: ", kv_plain)

		// add encrypted key-value pairs to the syncable object
		object_map["key_value_pairs"] = kv_string_encrypted
	} else {
		// add plaintext key-value pairs to the syncable object
		object_map["key_value_pairs"] = key_values_string
	}
	
	// insert into db
	dbc := session.DB("landline").C("SyncableObjects")
	err = dbc.Insert(object_map)
	if err != nil {
		panic(err)
	}
	
	// redirect
	c.Flash.Success("Object created")
	return c.Redirect(routes.SyncableObjects.ViewObject(object_key, object_map["uuid"].(string)))
}

func (c SyncableObjects) ViewObject(object_key, uuid string) revel.Result {
	
	// connect to mongodb
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()
	
	// query types
	schemasdb := session.DB("landline").C("Schemas")
	var schema map[string]interface{}
	err = schemasdb.Find(bson.M{"object_key": object_key}).One(&schema)
	if err != nil {
		revel.AppLog.Debug("Debug: ", err)
	}
	
	// query objects
	dbc := session.DB("landline").C("SyncableObjects")
	var object map[string]string
	err = dbc.Find(bson.M{"uuid": uuid, "object_type": object_key}).One(&object)
	if err != nil {
		revel.AppLog.Debug("Debug: ", err)
	}
	revel.AppLog.Debug("Debug: ", object)
	
	// find the deployment
	dbd := session.DB("landline").C("Deployments")
	var deployment map[string]string
	err = dbd.Find(bson.M{}).One(&deployment)

	kv_plain := object["key_value_pairs"]
	if deployment["enc_is_on"] == "True" {
		// decrypt key_value_pairs
		kv_hex, err := hex.DecodeString(object["key_value_pairs"])
		if err != nil {
			revel.AppLog.Debug("Debug: ", err)
		}
		kek_bytearr, ok := c.Session["kek"].([]byte)
		if ok != true {
			revel.AppLog.Debug("Error: ", "Type conversion")
		}
		kv_plain = string(cryptoWrapper.Decrypt(kek_bytearr, kv_hex))
		revel.AppLog.Debug("Debug: ", kv_plain)
	}
	
	// convert string of json to json to map
	byt := []byte((kv_plain))
	var key_values map[string]interface{}
	if err := json.Unmarshal(byt, &key_values); err != nil {
		panic(err)
	}
		
	return c.Render(object, key_values, schema)
}

func (c SyncableObjects) ListObjects(object_key string) revel.Result {
	
	// connect to mongodb
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()
	
	// query
	dbs := session.DB("landline").C("Schemas")
	
	var object_type map[string]interface{}
	err = dbs.Find(bson.M{"object_key": object_key}).One(&object_type)
	if err != nil {
		revel.AppLog.Debug("Debug: ", err)
	}
	
	
	dbc := session.DB("landline").C("SyncableObjects")
	
	var results []map[string]interface{}
	err = dbc.Find(bson.M{"object_type": object_key}).All(&results)
	if err != nil {
		revel.AppLog.Debug("Debug: ", err)
	}
	
	revel.AppLog.Debug("Debug: ", results)
	
	// decrypt each
	var object_list []map[string]interface{}
	for u, result := range results {
		revel.AppLog.Debug("Debug: ", u)

		// object that the client does not know about
		syncable_object_map := make(map[string]interface{})
		syncable_object_map["uuid"] = result["uuid"]
		syncable_object_map["object_type"] = result["object_type"]
		syncable_object_map["time_modified_since_creation"] = result["time_modified_since_creation"]
		
		// find the deployment
		dbd := session.DB("landline").C("Deployments")
		var deployment map[string]string
		err = dbd.Find(bson.M{}).One(&deployment)

		if deployment["enc_is_on"] == "True" {	
			// decrypt
			kv_hex, err := hex.DecodeString(result["key_value_pairs"].(string))
			if err != nil {
				revel.AppLog.Debug("Debug: ", err)
			}
			kek_bytearr, ok := c.Session["kek"].([]byte)
			if ok != true {
				revel.AppLog.Debug("Error: ", "Type conversion")
			}
			kv_plain := cryptoWrapper.Decrypt(kek_bytearr, kv_hex)
			
			// unmarshal the json
			var obj_json map[string]interface{}
			if err := json.Unmarshal(kv_plain, &obj_json); err != nil {
				panic(err)
			}

			syncable_object_map["key_value_pairs"] = obj_json
		} else {
			// unmarshal the json
			var obj_json map[string]interface{}
			if err := json.Unmarshal(result["key_value_pairs"].([]byte), &obj_json); err != nil {
				panic(err)
			}

			syncable_object_map["key_value_pairs"] = obj_json
		}

		object_list = append(object_list, syncable_object_map)
	}
		
	return c.Render(object_type, object_key, results, object_list)
}

func (c SyncableObjects) MarkdownEditor() revel.Result {
	return c.Render()
}










