package server

import "net/http"

func multiplexer() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /hms/dives", funcWithDataAccess(renderDives))
	trace(_https, "handler registered for /hms/dives")

	mux.HandleFunc("GET /hms/dives/{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/hms/dives", http.StatusMovedPermanently)
	})
	trace(_https, "handler registered for /hms/dives/")

	mux.HandleFunc("GET /hms/sites", funcWithDataAccess(renderSites))
	trace(_https, "handler registered for /hms/sites")

	mux.HandleFunc("GET /hms/sites/{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/hms/sites", http.StatusMovedPermanently)
	})
	trace(_https, "handler registered for /hms/sites/")

	mux.HandleFunc("GET /hms/tags", funcWithDataAccess(renderTags))
	trace(_https, "handler registered for /hms/tags")

	mux.HandleFunc("GET /hms/tags/{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/hms/tags", http.StatusMovedPermanently)
	})
	trace(_https, "handler registered for /hms/tags/")

	mux.HandleFunc("GET /hms/dives/{id}", funcWithDataAccess(renderDive))
	trace(_https, "handler registered for /hms/dives/{id}")

	mux.HandleFunc("GET /hms/sites/{id}", funcWithDataAccess(renderSite))
	trace(_https, "handler registered for /hms/sites/{id}")

	mux.HandleFunc("GET /hms/tags/{tag}", funcWithDataAccess(renderTaggedDives))
	trace(_https, "handler registered for /hms/tags/{tag}")

	mux.HandleFunc("GET /hms/about", func(w http.ResponseWriter, r *http.Request) {
		renderTemplate(w, Page{
			Title:      "this site",
			Supertitle: "about",
			About:      true,
		})
	})
	trace(_https, "handler registered for /hms/about")

	// data handlers
	mux.HandleFunc("GET /data/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	trace(_https, "handler registered for /data/")

	mux.HandleFunc("GET /data/sites", funcWithDataAccess(fetchSites))
	trace(_https, "handler registered for /data/sites")
	// DEVNOTE: /data/sites/{$} returns 404

	mux.HandleFunc("GET /data/sites/{id}", funcWithDataAccess(fetchSite))
	trace(_https, "handler registered for /data/sites/{id}")

	mux.HandleFunc("GET /data/trips", funcWithDataAccess(fetchTrips))
	trace(_https, "handler registered for /data/trips")
	// DEVNOTE: /data/trips/{$} returns 404

	mux.HandleFunc("GET /data/dives", funcWithDataAccess(fetchDives))
	trace(_https, "handler registered for /data/dives")
	// DEVNOTE: /data/dives/{$} returns 404

	mux.HandleFunc("GET /data/dives/{id}", funcWithDataAccess(fetchDive))
	trace(_https, "handler registered for /data/dives/{id}")

	mux.HandleFunc("GET /data/tags", funcWithDataAccess(fetchTags))
	trace(_https, "handler registered for /data/tags")
	// DEVNOTE: /data/tags/{$} returns 404

	mux.HandleFunc("GET /", defaultHandler)
	trace(_https, "handler registered for /")

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/hms/dives", http.StatusMovedPermanently)
	})
	trace(_https, "handler registered for /{$}")

	// local API handlers
	if _control_block.localAPI {
		mux.HandleFunc("GET /data/0", funcWithDataAccess(fetchAll))
		trace(_https, "handler registered for /data/0")

		mux.HandleFunc("POST /action/fail", forceFailure)
		trace(_https, "handler registered for /action/fail")

		mux.HandleFunc("POST /action/rebuild", rebuildDatabase)
		trace(_https, "handler registered for /action/rebuild")
	}

	return mux
}
