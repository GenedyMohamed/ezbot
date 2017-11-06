package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/ramin0/chatbot"
	"golang.org/x/net/html"
)

// Autoload environment variables in .env
import _ "github.com/joho/godotenv/autoload"

type Course struct {
	Name string
	Code string
	Link string
}

type File struct {
	Name string
	Link string
}

func removeSessionDups(session chatbot.Session) {

	result := []Course{}
	encountered := map[Course]bool{}
	courses := session["courses"].([]Course)

	for v := range courses {
		if encountered[courses[v]] == false {
			encountered[courses[v]] = true
			result = append(result, courses[v])
		}
	}
	session["courses"] = result

	/*for i := 0; i < len(courses); i++ {
		for j := i + 1; j <= len(courses); j++ {

			if (courses[i].Code == courses[j].Code) {
					fmt.Println(i,j,length,"fufhufhfrurhugrhugthigrhitghighgi")
				courses[j] = courses[length]
				courses = courses[0:length]
				length--
				j--
			}
		}
	}*/

}

func courses(session chatbot.Session) {
	var courses []Course
	session["courses"] = courses
	//session["map"] = make(map[string]string)
	url := "http://met.guc.edu.eg/Courses/Undergrad.aspx"
	resp, _ := http.Get(url)
	r, _ := regexp.Compile("^[A-Z]{4}[ ][0-9]{3}$")
	defer resp.Body.Close()
	z := html.NewTokenizer(resp.Body)
	for {
		tt := z.Next()
		switch {
		case tt == html.ErrorToken:
			// End   of the  document, we're done
			removeSessionDups(session)
			return
		case tt == html.StartTagToken:
			t := z.Token()
			isAnchor := t.Data == "a"
			if isAnchor {
				link := t.Attr
				z.Next()
				data := z.Token().Data
				if len(data) > 8 && r.MatchString(data[0:8]) {
					for _, a := range link {
						if a.Key == "href" {
							courseLink := a.Val[27:]
							session["courses"] = append(session["courses"].([]Course), Course{data, data[0:8], courseLink})
							break
						}
					}

					//session["map"].(map[string]string)[data[0:8]] = data
				}
			}
		}
	}
}

// func navigate(session chatbot.Session, courseCode string) {
// 	url := "http://met.guc.edu.eg/Courses/Undergrad.aspx"
// 	resp, _ := http.Get(url)

// 	defer resp.Body.Close()
// 	z := html.NewTokenizer(resp.Body)
// 	found := false
// 	for {
// 		tt := z.Next()
// 		switch {
// 		case tt == html.ErrorToken:
// 			// End   of the  document, we're done
// 			return
// 		case tt == html.StartTagToken:
// 			t := z.Token()
// 			isAnchor := t.Data == "a"
// 			if isAnchor {
// 				link := t.Attr
// 				z.Next()
// 				data := z.Token().Data
// 				if len(data) > 8 && data[0:8] == courseCode {
// 					for _, a := range link {
// 						if a.Key == "href" {
// 							session["courseID"] = a.Val[27:]
// 							found = true
// 							break
// 						}
// 					}
// 				}
// 			}
// 		}
// 		if found {
// 			break
// 		}
// 	}
// }

func files(session chatbot.Session, id string) {
	//session["files"] = make(map[string]string)
	var files []File
	session["files"] = files
	url := "http://met.guc.edu.eg/Courses/Material.aspx?crsEdId=" + id
	resp, _ := http.Get(url)
	//r, _ := regexp.Compile("^[A-Z]{4}[ ][0-9]{3,}$")
	defer resp.Body.Close()
	z := html.NewTokenizer(resp.Body)
	for {
		tt := z.Next()
		switch {
		case tt == html.ErrorToken:
			// End  of the  document, we're done
			return
		case tt == html.StartTagToken:
			t := z.Token()
			isAnchor := t.Data == "a"
			if isAnchor {
				z.Next()
				data := z.Token().Data // name of the file
				for _, attr := range t.Attr {
					if attr.Key == "href" {
						if strings.Contains(attr.Val, "Download") {
							//session["files"].(map[string]string)[data] = "met.guc.edu.eg" + attr.Val[2:]
							session["files"] = append(session["files"].([]File), File{data, "met.guc.edu.eg" + attr.Val[2:]})
						}
						break
					}
				}
			}
		}
	}
}

func announce(session chatbot.Session, id string) {
	a := ParseRSSMetCourseFeed(id) // a = array of Announcement objects: Annoucement{Title, Description}
	//result := ""
	var buffer bytes.Buffer
	url := "http://met.guc.edu.eg/Courses/CourseEdition.aspx?crsEdId=" + id
	resp, _ := http.Get(url)
	defer resp.Body.Close()
	z := html.NewTokenizer(resp.Body)
	for {
		tt := z.Next()
		switch {
		case tt == html.ErrorToken:
			//session["announcements"] = result
			if len(buffer.String()) < 3 {
				session["announcements"] = "No announcements found."
			} else {
				session["announcements"] = buffer.String()
			}
			return
		case tt == html.TextToken:
			text := string(z.Text())
			for _, announcement := range a {
				if text == announcement.Title[15:] {
					//result += text + "\n" + announcement.Description + "\n"
					buffer.WriteString(text + ": ")
					buffer.WriteString(announcement.Description + " | ")
					// 					a[len(a)-1], a[i] = a[i], a[len(a)-1]
					//     			a = a[:len(a)-1]
				}
			}

		}
	}
}

// phase nil: user has supplied name -> ask them what they require (announcements or files)
// phase 10: user has specified announcements -> ask for which subject
// 		phase 101: user has given valid subject -> retrieve announcements for given subject and send to user 										END_OF_THREAD
// 		phase 102: user has given invalid subject -> show list of subjects and wait till user enters valid subject code
// phase 21: user has specified announcements AND given subject code
// 		phase 211: user has

func chatbotProcess(session chatbot.Session, message string) (string, error) {

	switch {
	//case session["presetMessage"] != nil:
	//return session["presetMessage"].(string), nil

	case session["phase"] == nil:
		go courses(session)
		session["phase"] = "announcement or file"
		return fmt.Sprintf("Hi " + message + ", nice to meet you. Would you like me to fetch you announcements for a course or a certain file?"), nil

	case session["phase"] == "announcement or file":
		if strings.Contains(strings.ToLower(message), "announce") || strings.Contains(strings.ToLower(message), "view") {
			session["phase"] = "announcement. which course"
			return "Ok, for which course?", nil
		}
		if strings.Contains(strings.ToLower(message), "file") || strings.Contains(strings.ToLower(message), "download") {
			session["phase"] = "file. which course"
			return "Ok, for which course?", nil
		}
		return "Please specify whether you need announcements or files.", nil

	case session["phase"] == "announcement. which course":
		r, _ := regexp.Compile("[A-Z]{4}[ ][0-9]{3}")
		code1 := r.FindAllString(strings.ToUpper(message), 1)
		r2, _ := regexp.Compile("[A-Z]{4}[0-9]{3}")
		code2 := r2.FindAllString(strings.ToUpper(message), 1)
		//courseList := session["map"].(map[string]string)
		courseList := session["courses"].([]Course)
		//found := false

		if code1 != nil {
			for _, course := range courseList {
				if course.Code == code1[0] {
					//found = true
					//navigate(session, course)
					//id := session["courseID"].(string)
					announce(session, course.Link)
					session["phase"] = "announcement or file"
					return session["announcements"].(string) + " Would you like to fetch another file or announcement?", nil
				}
			}
		}
		if code2 != nil {
			newcode := code2[0][:4] + " " + code2[0][4:]
			for _, course := range courseList {
				if course.Code == newcode {
					//found = true
					//navigate(session, code)
					//id := session["courseID"].(string)
					announce(session, course.Link) // TODO  session["announcements"] = blla bla bla OR no announcements found
					session["phase"] = "announcement or file"
					return session["announcements"].(string) + " Would you like to fetch another file or announcement?", nil
				}
			}
		}
		//if !found {
		var buffer bytes.Buffer
		for _, course := range courseList {
			buffer.WriteString(course.Name + ", ")
			//buffer.WriteString(", ")
		}
		return fmt.Sprintf("Invalid course code. Did you mean one of the following?\n" + buffer.String()), nil
		//}

	case session["phase"] == "file. which course":
		r, _ := regexp.Compile("[A-Z]{4}[ ][0-9]{3}")
		code1 := r.FindAllString(strings.ToUpper(message), 1)
		r2, _ := regexp.Compile("[A-Z]{4}[0-9]{3}")
		code2 := r2.FindAllString(strings.ToUpper(message), 1)
		//courseList := session["map"].(map[string]string)
		courseList := session["courses"].([]Course)
		//found := false

		if code1 != nil {
			for _, course := range courseList {
				if course.Code == code1[0] {
					//found = true
					//navigate(session, code)
					//id := session["courseID"].(string)
					files(session, course.Link)
					session["phase"] = "which file"
					f := session["files"].([]File)
					var buffer bytes.Buffer
					for _, file := range f {
						buffer.WriteString(file.Name + ", ")
						//buffer.WriteString(", ")
					}
					return "Which file would you like for " + course.Code + "? These are the available files." + buffer.String(), nil
				}
			}
		}
		if code2 != nil {
			newcode := code2[0][:4] + " " + code2[0][4:]
			for _, course := range courseList {
				if course.Code == newcode {
					//found = true
					//navigate(session, code)
					//id := session["courseID"].(string)
					files(session, course.Link)
					session["phase"] = "which file"
					f := session["files"].([]File)
					var buffer bytes.Buffer
					for _, file := range f {
						buffer.WriteString(file.Name + ", ")
						//buffer.WriteString(", ")
					}
					return "Which file would you like for " + course.Code + "? These are the available files." + buffer.String(), nil
				}
			}
		}
		//if !found {
		var buffer bytes.Buffer
		for _, course := range courseList {
			buffer.WriteString(course.Name + ", ")
			//buffer.WriteString(", ")
		}
		return fmt.Sprintf("Invalid course code. Did you mean one of the following?\n" + buffer.String()), nil
		//}

	case session["phase"] == "which file":

		f := session["files"].([]File)
		var buffer bytes.Buffer
		for _, file := range f {
			if strings.Contains(strings.ToLower(file.Name), strings.ToLower(message)) {
				session["phase"] = "announcement or file"
				//session["presetMessage"] = "What do you need now " + session["name"].(string) + "? Announcements or files?"
				if len(message) > 4 {
					return "The download link for " + file.Name + " is " + file.Link + ". Would you like to fetch another file or announcements?", nil
				}
			}
			buffer.WriteString(file.Name + ", ")
			//buffer.WriteString(", ")
		}

		return fmt.Sprintf("File not found. Did you mean one of the following?\n" + buffer.String()), nil

	}
	return "anything", nil
}

func main() {
	// Uncomment the following lines to customize the chatbot
	chatbot.WelcomeMessage = "Hello. I'm EzBot. I'm here to keep you up-to-date with your courses at MET!\nWhat's your name?"
	chatbot.ProcessFunc(chatbotProcess)

	// Use the PORT environment variable
	port := os.Getenv("PORT")
	// Default to 3000 if no PORT environment variable was defined
	if port == "" {
		port = "3000"
	}

	// Start the server
	fmt.Printf("Listening on port %s...\n", port)
	log.Fatalln(chatbot.Engage(":" + port))
}

// func chatbotProcess(session chatbot.Session, message string) (string, error) {
// 	sen := strings.Split(message, " ")
// 	name := sen[len(sen)-1]
// 	if (session["name"] == nil){
// 	if (strings.Contains(strings.ToLower(name), "hi") || strings.Contains(strings.ToLower(name), "hello") || strings.Contains(strings.ToLower(name), "hey")){
// 		if (len(sen) < 3){
// 			return fmt.Sprintf("Hi, What is your name?"), nil
// 		}else{
// 			session["name"] = name
// 			go courses(session)
// 			go files(session, "http://met.guc.edu.eg/Courses/Material.aspx?crsEdId=759")
// 			return fmt.Sprintf("Hello "+name+"! How can I help you?"), nil
// 		}
// 	}else{
// 		if (len(sen) == 2){
// 			name = sen[len(sen)-2]+" "+sen[len(sen)-1]
// 			session["name"] = name
// 			go courses(session)
// 			go files(session, "http://met.guc.edu.eg/Courses/Material.aspx?crsEdId=759")
// 			return fmt.Sprintf("Hello "+name+"! How can I help you?"), nil
// 		}else{
// 			session["name"] = name
// 			go courses(session)
// 			go files(session, "http://met.guc.edu.eg/Courses/Material.aspx?crsEdId=759")
// 			return fmt.Sprintf("Hello "+name+"! How can I help you?"), nil
// 		}
// 	}
// 	}else{
// 		if (strings.Contains(strings.ToLower(message), "thnx") || strings.Contains(strings.ToLower(message), "thank")){
// 			return fmt.Sprintf("You are mostly welcomed anytime :)"), nil
// 		}
// 		if (strings.Contains(strings.ToLower(message), "help")){
// 			return fmt.Sprintf("For a course, you can check quiz timings, announcements or download a file"), nil
// 		}
// 		if (strings.Contains(strings.ToLower(message), "list")){
// // 			strings.Split(session["data"], "|")
// 			//return fmt.Sprintf(session["data"].(string)), nil
// 			m := session["map"].(map[string]string)
// 			var buffer bytes.Buffer

// 			for _, v := range m{
// 				buffer.WriteString(v)
// 				buffer.WriteString(", ")
// 			}
// 			return fmt.Sprintf(buffer.String()), nil
// 		}
// 		if (strings.Contains(strings.ToLower(message), "files")){
// // 			strings.Split(session["data"], "|")
// 			//return fmt.Sprintf(session["data"].(string)), nil
// 			if(session["files"] == nil){
// 				return fmt.Sprintf("error sorry"), nil
// 			}
// 			f := session["files"].(map[string]string)
// 			var buffer bytes.Buffer

// 			for filename, _ := range f{
// 				buffer.WriteString(filename)
// 				buffer.WriteString(", ")
// 			}
// 			return fmt.Sprintf(buffer.String()), nil
// 		}

// 		if (strings.Contains(strings.ToLower(message), "show") || strings.Contains(strings.ToLower(message), "announc")){
// 			r, _ := regexp.Compile("[A-Z]{4}[ ][0-9]{3}")
// 			code := r.FindAllString(message, 1)
// 			r2, _ := regexp.Compile("[A-Z]{4}[0-9]{3}")
// 			code2 := r2.FindAllString(message, 1)

// 			if (code == nil){
// 				if (code2 == nil){
// 				return fmt.Sprintf("For which course?"), nil
// 				}else{
// 					newcode := code2[0][:4]+" "+code2[0][4:]
// 					navigate(session, newcode)
// 					return fmt.Sprintf(session["courseID"].(string)), nil
// 				}
// 			}else{
// 				navigate(session, code[0])
// 				return fmt.Sprintf(session["courseID"].(string)), nil
// 			}

// 		}

// 		return fmt.Sprintf("Bye! Was not fun talking to you :)"), nil
// 	}
// }
