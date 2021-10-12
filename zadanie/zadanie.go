package main

import (
	"context"
	"fmt"
	"github.com/IncSW/geoip2"
	"github.com/likexian/whois-go"
	whoisparser "github.com/likexian/whois-parser-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"io/ioutil"
	"net"
	"strings"
	"time"
)





type Data struct {
	ID primitive.ObjectID `bson:"_id, omitempty"`
	Host string `json:Host,omitempty`
	IP []string  `json:IP,omitempty`
	Asn uint32   `json: ASN,omitempty`
	GeoIP string  `json: GeoIP,omitempty`
	WhoISInformation map[string]interface{} `json:WhoISInformation,omitempty`
	RegisterTime time.Time `json:RegisterTime,omitempty`

}

type UpdateData struct {
	UpdateIP []string  `json:UpdateIP,omitempty`
	UpdateAsn uint32   `json: UpdateASN,omitempty`
	UpdateGeoIP string  `json: UpdateGeoIP,omitempty`
	UpdateWhoISInformation map[string]interface{} `json:UpdateWhoISInformation,omitempty`
	UpdateRegisterTime time.Time `json:UpdateRegisterTime,omitempty`

}

type Pars struct{
	pars []string
}


func typeof(v interface{}) string {
	return fmt.Sprintf("%T", v)
}

func (info *Data)getIP(host string) []string{
	info.Host=host
	info.IP=make([]string,0,2)
	ip,err:=net.LookupIP(host)
	if err!=nil{
		fmt.Errorf("some error %v", err)
	}
	for _,j:=range ip{
		info.IP = append(info.IP, j.String())
	}
	return info.IP
}

//https://github.com/IncSW/geoip2 docs
//https://dev.maxmind.com/geoip/geoip2/downloadable/
func (info *Data)getGeo2(ip string) (string, uint32, error) {
	reader, err := geoip2.NewCityReaderFromFile("/GeoLite2-City.mmdb")
	if err != nil {
		fmt.Println("some error %v", err)

	}
	record, err := reader.Lookup(net.ParseIP(ip))
	if err != nil {
		fmt.Println("some error %v", err)
	}
	info.Asn=record.City.GeoNameID
	info.GeoIP=record.Country.ISOCode
	return info.GeoIP,info.Asn,nil
}

//WhoIS inform
func (info *Data)whoIS(host string) map[string]interface{}{
	parseInform:=make(map[string]interface{})
	query,err:=whois.Whois(host)
	if err!=nil{
		fmt.Errorf("some error %v", err)
	}

	result,err:=whoisparser.Parse(query)
	if err!=nil{
		fmt.Errorf("some error %v", err)
	}

	parseInform["Information about Domain"]=result.Domain
	parseInform["Information about Registrar"]=result.Registrar
	parseInform["Information about Registrant"]=result.Registrant
	parseInform["Information about Billing"]=result.Billing
	parseInform["Information about Technical"]=result.Technical
	parseInform["Information about Administrative"]=result.Administrative
	info.WhoISInformation = parseInform
	return info.WhoISInformation
}

func (info *Data)GetRegisterTime() time.Time{
	info.RegisterTime=time.Now()
	return info.RegisterTime
}

func (info *Data)GetID() primitive.ObjectID{
	info.ID=primitive.NewObjectID()
	return info.ID

}

func  main() {
	var data Data
	var updateData UpdateData
	Client, err := mongo.NewClient(options.Client().ApplyURI("mongodb+srv://username.mongodb.net/zadanie?retryWrites=true&w=majority", ))
	if err != nil {
		fmt.Errorf("some error %v", err)
	}

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = Client.Connect(ctx)
	if err != nil {
		fmt.Errorf("some error %v", err)
	}
	defer Client.Disconnect(ctx)
	Collection := Client.Database("my_database").Collection("Information about host")

	getPostId:=func(id primitive.ObjectID) []bson.M {
		filter := bson.D{{"host",data.Host}}
		cursor,err := Collection.Find(context.TODO(), filter)
		if err != nil {
			fmt.Errorf("some error %v", err)
		}
		var results []bson.M
		if err = cursor.All(context.TODO(), &results); err != nil {
			fmt.Errorf("some error %v", err)
		}
		return results

	}
	sendPost := func(Id primitive.ObjectID,Host string, IP []string, Asn uint32, GeoIP string, WhoISInformation map[string]interface{}, RegisterTime time.Time)  {

		post := Data{Id,Host, IP, Asn, GeoIP, WhoISInformation, RegisterTime}
		InsertResult, err := Collection.InsertOne(context.TODO(), post)
		if err != nil {
			fmt.Errorf("some error %v", err)
		}
		fmt.Println("Inserted post with ID:", InsertResult.InsertedID)
	}
	insertIntoHost:= func(IP []string, Asn uint32, GeoIP string, WhoISInformation map[string]interface{}, RegisterTime time.Time){
		filter := bson.D{{"host",data.Host}}
		document:=bson.D{{"$set", bson.D{{"updateip", updateData.UpdateIP},
				{"updateasn",updateData.UpdateAsn},
				{"updategeoip", updateData.UpdateGeoIP},
				{"updatewhoisinformation", updateData.UpdateWhoISInformation},
				{"updateregistertime",updateData.UpdateGeoIP},
			}}}

		opts := options.Update().SetUpsert(true)
		result, err:=Collection.UpdateMany(context.TODO(),filter, document, opts)
		if err!=nil{
			fmt.Errorf("Something went wrong %v", err)
		}
		if result.MatchedCount != 0 {
			fmt.Println("matched and replaced an existing document")
			return
		}
		if result.UpsertedCount != 0 {
			fmt.Printf("inserted a new document with ID %v\n", result.UpsertedID)
		}

	}
	txtF,err:=ioutil.ReadFile("/parser/results/2020-10-08/1.txt")
	readData:=strings.Split(string(txtF),",")
	for _,host:=range readData{
		host=strings.ReplaceAll(host,"https://www.cubdomain.com/site/","")
		host=strings.ReplaceAll(host," ","")
		host=strings.Trim(host,"\"")

		data.ID=data.GetID()
		data.WhoISInformation = data.whoIS(host)
		data.Host = host
		data.RegisterTime = data.GetRegisterTime()
		ip := data.getIP(host)
		fmt.Println(ip)
		if len(ip)==0{
			fmt.Println("Can't resolve IP address")
			data.IP=[]string{"Can't resolve IP address"}
			data.Asn=0
			data.GeoIP="None"

		}else{
			data.IP = data.getIP(host)
			for _, i := range ip[:1] {
				data.GeoIP, data.Asn ,err= data.getGeo2(i)
				if err !=nil{
				}
			}
		}


		similarHosts:=[]bson.M{}
		similarHosts=getPostId(data.ID)
		if len(similarHosts)>=1{
			fmt.Println("Update post")
			updateData.UpdateWhoISInformation = data.whoIS(host)
			updateData.UpdateRegisterTime = data.GetRegisterTime()
			ip := data.getIP(host)
			if len(ip)==0{
				fmt.Println("Can't resolve IP address")
				updateData.UpdateIP=[]string{"Can't resolve IP address"}
				updateData.UpdateAsn=0
				updateData.UpdateGeoIP="None"
			}else{
				updateData.UpdateIP = data.getIP(host)
				for _, i := range ip[:1] {
					updateData.UpdateGeoIP, updateData.UpdateAsn,err = data.getGeo2(i)
					if err!=nil {
						continue
					}
				}
			}
			insertIntoHost(updateData.UpdateIP, updateData.UpdateAsn,updateData.UpdateGeoIP,updateData.UpdateWhoISInformation,updateData.UpdateRegisterTime)
		}else{
			fmt.Println("Those post isn't exist\nSend new post")
			sendPost(data.ID,data.Host,data.IP,data.Asn,data.GeoIP,data.WhoISInformation, data.RegisterTime)
		}


	}
}
