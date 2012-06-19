package main

import (
	"code.google.com/p/gorilla/pat"
	"code.google.com/p/gorilla/sessions"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"thegoods.biz/tmplmgr"
	"worker"
)

const (
	appname   = "goci"
	store_key = "foobar"
)

var (
	mode = tmplmgr.Production

	//TODO: make sure these things happen after the env import
	assets_dir    = filepath.Join(env("APPROOT", ""), "assets")
	template_dir  = filepath.Join(env("APPROOT", ""), "templates")
	dist_dir      = filepath.Join(env("APPROOT", ""), "dist")
	base_template = tmplmgr.Parse(tmpl_root("base.tmpl"))

	store     = sessions.NewCookieStore([]byte(store_key))
	base_meta = &Meta{
		CSS: list{
			"bootstrap-superhero.min.css",
			"bootstrap-responsive.min.css",
			"main.css",
		},
		JS: list{
			"jquery.min.js",
			"jquery-ui.min.js",
			"bootstrap.js",
		},
		BaseTitle: "GoCI",
	}
	router = pat.New()
)

func main() {
	//revert to development mode if debug is set
	if env("DEBUG", "") != "" {
		mode = tmplmgr.Development
	}

	//set our compiler mode
	tmplmgr.CompileMode(mode)

	//add blocks to base template
	base_template.Blocks(tmpl_root("*.block"))
	base_template.Call("reverse", reverse)

	//get our mongo credentials
	var db_name, db_path = appname, "localhost"
	if conf := env("MONGOLAB_URI", ""); conf != "" {
		db_path = conf
		parsed, err := url.Parse(conf)
		if err != nil {
			log.Fatal("Error parsing DATABASE_URL: %q: %s", conf, err)
		}
		db_name = parsed.Path[1:]
	}
	log.Printf("\tdb_path: %s\n\tdb_name: %s", db_path, db_name)

	//build our config
	config := worker.Config{
		Debug:  env("DEBUG", "") != "",
		App:    need_env("APPNAME"),
		Api:    need_env("APIKEY"),
		Name:   db_name,
		URL:    db_path,
		GOROOT: need_env("GOROOT"),
		Host:   need_env("HOST"),
	}

	//run the worker setup
	go func() {
		if err := worker.Setup(config); err != nil {
			log.Fatal("error during setup:", err)
		}
		log.Print("setup complete")
	}()

	//set up our handlers
	handleGet("/bins/{id}", handlerFunc(handle_test_request), "test_request")
	handlePost("/bins/{id}/err", handlerFunc(handle_test_error), "test_error") //more specific one has to be listed first
	handlePost("/bins/{id}", handlerFunc(handle_test_response), "test_response")

	handleGet("/build/{id}", handlerFunc(handle_build_info), "build_info")

	handlePost("/hooks/github/package", handlerFunc(handle_github_hook_package), "github_hook_package")
	handlePost("/hooks/github/workspace", handlerFunc(handle_github_hook_workspace), "github_hook_workspace")
	handlePost("/hooks/bitbucket/package", handlerFunc(handle_bitbucket_hook_package), "bitbucket_hook_package")
	handlePost("/hooks/bitbucket/workspace", handlerFunc(handle_bitbucket_hook_workspace), "bitbucket_hook_workspace")
	handlePost("/hooks/google/package/{vcs}", handlerFunc(handle_google_hook_package), "google_hook_package")
	handlePost("/hooks/google/workspace/{vcs}", handlerFunc(handle_google_hook_workspace), "google_hook_workspace")

	handleGet("/how", handlerFunc(handle_how), "how")

	//debug handler
	handleRequest("/foo", handlerFunc(handle_simple_work), "foo")

	//add our index with 404 support
	handleRequest("/", handlerFunc(handle_index), "index")

	//build the nav and subnav
	base_meta.Nav = navList{
		&navBase{"Recent", reverse("index"), nil},
		// &navBase{"Projects", reverse("index"), nil},
		&navBase{"How", reverse("how"), nil},
	}
	base_meta.SubNav = navList{}

	//set up our router
	http.Handle("/", router)
	serve_static("/assets", asset_root(""))
	if err := http.ListenAndServe(":"+env("PORT", "9080"), nil); err != nil {
		log.Fatal(err)
	}
}
