you just accept that interface in your new instance function
store it in your migrate
and when you log out you just call that logger
that way the end user can customize the logging how they want to

and you would replace all these instances 
                    color.Red.Print("Failure:    ")
                    fmt.Println(v.name)
                    color.Red.Println(err)
with something like m.logger.Println(....)
or something

https://www.honeybadger.io/blog/golang-logging/