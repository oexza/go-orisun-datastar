package httpui

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/oexza/go-orisun-datastar/internal/eventstore"
	"github.com/oexza/go-orisun-datastar/internal/features/todo"
	"github.com/oexza/go-orisun-datastar/internal/views"
)

func (s Server) todoRoutes(r chi.Router) {
	r.Get("/todos", s.todosPage)
	r.Get("/todos/stream", s.todosStream)
	r.Post("/todos", s.createTodo)
	r.Post("/todos/{todoID}/rename", s.renameTodo)
	r.Post("/todos/{todoID}/complete", s.completeTodo)
	r.Post("/todos/{todoID}/reopen", s.reopenTodo)
	r.Post("/todos/{todoID}/delete", s.deleteTodo)
}

func (s Server) todosPage(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	todos, err := s.Todos.List(r.Context(), user.UserRegisteredID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = views.TodosPage(user, todos).Render(r.Context(), w)
}

func (s Server) todosStream(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	sse := datastar.NewSSE(w, r)
	ctx := r.Context()

	updates := make(chan struct{}, 1)
	notify := func() {
		select {
		case updates <- struct{}{}:
		default:
		}
	}

	sub, err := s.Subscriber.Subscribe(ctx, todo.Channel(user.UserRegisteredID), func(context.Context, []byte) {
		notify()
	})
	if err != nil {
		return
	}
	defer sub.Close()

	notify()

	for {
		select {
		case <-ctx.Done():
			return
		case <-updates:
			todos, err := s.Todos.List(ctx, user.UserRegisteredID)
			if err != nil {
				_ = alert(sse, err.Error())
				return
			}
			if err := sse.PatchElementTempl(views.TodoList(todos), datastar.WithSelector("#todo-list"), datastar.WithMode(datastar.ElementPatchModeInner)); err != nil {
				return
			}
		}
	}
}

func (s Server) createTodo(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	user := currentUser(r)
	_, err := s.Todos.CreateWithMetadata(r.Context(), user.UserRegisteredID, r.FormValue("title"), eventstore.HTTPCommandMetadata(r, user.UserRegisteredID))
	if err != nil {
		writeSSE(w, r, func(sse *datastar.ServerSentEventGenerator) error { return flashError(sse, err.Error()) })
		return
	}
	writeSSE(w, r, func(sse *datastar.ServerSentEventGenerator) error { return clearNewTodoTitle(sse) })
}

func (s Server) renameTodo(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	user := currentUser(r)
	emptySSE(w, r, s.Todos.RenameWithMetadata(r.Context(), user.UserRegisteredID, chi.URLParam(r, "todoID"), r.FormValue("title"), eventstore.HTTPCommandMetadata(r, user.UserRegisteredID)))
}

func (s Server) completeTodo(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	emptySSE(w, r, s.Todos.CompleteWithMetadata(r.Context(), user.UserRegisteredID, chi.URLParam(r, "todoID"), eventstore.HTTPCommandMetadata(r, user.UserRegisteredID)))
}

func (s Server) reopenTodo(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	emptySSE(w, r, s.Todos.ReopenWithMetadata(r.Context(), user.UserRegisteredID, chi.URLParam(r, "todoID"), eventstore.HTTPCommandMetadata(r, user.UserRegisteredID)))
}

func (s Server) deleteTodo(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	emptySSE(w, r, s.Todos.DeleteWithMetadata(r.Context(), user.UserRegisteredID, chi.URLParam(r, "todoID"), eventstore.HTTPCommandMetadata(r, user.UserRegisteredID)))
}
