package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"crypto/sha256"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

const (
	ImgDir = "images"
)

type Response struct {
	Message string `json:"message"`
}

type Items struct {
	Items []Item `json:"items"`
}

type Item struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Image string `json:"image"`
}

func root(c echo.Context) error {
	res := Response{Message: "Hello, world!"}
	return c.JSON(http.StatusOK, res)
}

/* 3.4 
curl -X POST 
--url http://localhost:9000/items 
-F name=jacket 
-F category=fashion 
-F image=@./local_image.jpg
*/

//ハッシュ化
func getSHA256Binary(s string) []byte {
    r := sha256.Sum256([]byte(s))
    return r[:]
}

func addItem(c echo.Context) error {
	// Get form data
	name := c.FormValue("name")
	category := c.FormValue("category")
	image, err := c.FormFile("image")
	if err != nil {
		return err
	}

	//hashしたものに".jpg"をつける
	hashedImg := getSHA256Binary(image.Filename)
	image_filename := fmt.Sprintf("%x%s",hashedImg , ".jpg")

	c.Logger().Infof("Receive item: %s", name)
	c.Logger().Infof("Receive item: %s", category)
	c.Logger().Infof("Receive item: %s", image.Filename)

	// 新しいitemを作る
	var newItem Item
	newItem.Name = name
	newItem.Category = category
	newItem.Image = image_filename
	
	// fileを開いて読んでitemsをゲットする
	jsonFile, err := os.Open("items.json")
	if err != nil {
		fmt.Println("JSONファイルを開けません", err)
		return c.JSON(http.StatusInternalServerError, err)
	}
	defer jsonFile.Close()
	jsonData, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		fmt.Println("JSONファイルを開けません", err)
		return c.JSON(http.StatusInternalServerError, err)
	}

	var items Items
	
	json.Unmarshal(jsonData, &items)


	// fileに追加
	items.Items = append(items.Items, newItem)
	file, _ := json.MarshalIndent(items, "", " ")
	_ = ioutil.WriteFile("items.json", file, 0644)

	message := fmt.Sprintf("item received: %s", name)
	res := Response{Message: message}

	return c.JSON(http.StatusOK, res)
}

func getItem(c echo.Context) error {
	jsonFile, err := os.Open("items.json")
	if err != nil {
		fmt.Println("JSONファイルを開けません", err)
		return c.JSON(http.StatusInternalServerError, err)
	}
	defer jsonFile.Close()
	jsonData, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		fmt.Println("JSONファイルを開けません", err)
		return c.JSON(http.StatusInternalServerError, err)
	}

	var items Items
	json.Unmarshal(jsonData, &items)

	return c.JSON(http.StatusOK, items) 
}

func getImg(c echo.Context) error {
	// Create image path
	imgPath := path.Join(ImgDir, c.Param("imageFilename"))

	if !strings.HasSuffix(imgPath, ".jpg") {
		res := Response{Message: "Image path does not end with .jpg"}
		return c.JSON(http.StatusBadRequest, res)
	}
	if _, err := os.Stat(imgPath); err != nil {
		c.Logger().Debugf("Image not found: %s", imgPath)
		imgPath = path.Join(ImgDir, "default.jpg")
	}
	return c.File(imgPath)
}


func getItemByID(c echo.Context)  error{
	jsonFile, err := os.Open("items.json")
	if err != nil {
		fmt.Println("JSONファイルを開けません", err)
		return c.JSON(http.StatusInternalServerError, err)
	}
	defer jsonFile.Close()
	jsonData, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		fmt.Println("JSONファイルを開けません", err)
		return c.JSON(http.StatusInternalServerError, err)
	}

	var items Items
	json.Unmarshal(jsonData, &items)

	id := c.Param("item_id") 
	
	idItem := Item{}
	for index, element := range items.Items {
		if(strconv.Itoa(index) == id){
			idItem = element
			return c.JSON(http.StatusOK, idItem)
		}
	}
	res := Response{Message: "Not found"}
	return c.JSON(http.StatusNotFound, res)
}


func main() {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Logger.SetLevel(log.INFO)

	front_url := os.Getenv("FRONT_URL")
	if front_url == "" {
		front_url = "http://localhost:3000"
	}
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{front_url},
		AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete},
	}))

	// Routes
	e.GET("/", root)
	e.POST("/items", addItem)
	e.GET("/items", getItem)
	e.GET("/image/:imageFilename", getImg)
	e.GET("/items/:item_id", getItemByID)

	// Start server
	e.Logger.Fatal(e.Start(":9000"))
}
