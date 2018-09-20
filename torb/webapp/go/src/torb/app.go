package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/errgroup"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/middleware"

	"github.com/najeira/measure"
	echopprof "github.com/sevenNt/echo-pprof"
)

type User struct {
	ID        int64  `json:"id,omitempty"`
	Nickname  string `json:"nickname,omitempty"`
	LoginName string `json:"login_name,omitempty"`
	PassHash  string `json:"pass_hash,omitempty"`
}

type Event struct {
	ID       int64  `json:"id,omitempty"`
	Title    string `json:"title,omitempty"`
	PublicFg bool   `json:"public,omitempty"`
	ClosedFg bool   `json:"closed,omitempty"`
	Price    int64  `json:"price,omitempty"`

	Total   int                `json:"total"`
	Remains int                `json:"remains"`
	Sheets  map[string]*Sheets `json:"sheets,omitempty"`
}

type Sheets struct {
	Total   int      `json:"total"`
	Remains int      `json:"remains"`
	Detail  []*Sheet `json:"detail,omitempty"`
	Price   int64    `json:"price"`
}

type Sheet struct {
	ID    int64  `json:"-"`
	Rank  string `json:"-"`
	Num   int64  `json:"num"`
	Price int64  `json:"-"`

	Mine           bool       `json:"mine,omitempty"`
	Reserved       bool       `json:"reserved,omitempty"`
	ReservedAt     *time.Time `json:"-"`
	ReservedAtUnix int64      `json:"reserved_at,omitempty"`
}

type Reservation struct {
	ID         int64      `json:"id"`
	EventID    int64      `json:"-"`
	SheetID    int64      `json:"-"`
	UserID     int64      `json:"-"`
	ReservedAt *time.Time `json:"-"`
	CanceledAt *time.Time `json:"-"`
	EventPrice int64      `json:"-"`

	Event          *Event `json:"event,omitempty"`
	SheetRank      string `json:"sheet_rank,omitempty"`
	SheetNum       int64  `json:"sheet_num,omitempty"`
	Price          int64  `json:"price,omitempty"`
	ReservedAtUnix int64  `json:"reserved_at,omitempty"`
	CanceledAtUnix int64  `json:"canceled_at,omitempty"`
}

type Administrator struct {
	ID        int64  `json:"id,omitempty"`
	Nickname  string `json:"nickname,omitempty"`
	LoginName string `json:"login_name,omitempty"`
	PassHash  string `json:"pass_hash,omitempty"`
}

func sheetIDtoSheet(id int64) Sheet {
	/*

		+-----+------+-----+-------+
		| id  | rank | num | price |
		+-----+------+-----+-------+
		| 501 | C    |   1 |     0 |
		| 201 | B    |   1 |  1000 |
		|  51 | A    |   1 |  3000 |
		|   1 | S    |   1 |  5000 |
		+-----+------+-----+-------+
	*/
	if id > 500 {
		return Sheet{ID: id, Rank: "C", Num: id - 500, Price: 0}
	}
	if id > 200 {
		return Sheet{ID: id, Rank: "B", Num: id - 200, Price: 1000}
	}
	if id > 50 {
		return Sheet{ID: id, Rank: "A", Num: id - 50, Price: 3000}
	}
	return Sheet{ID: id, Rank: "S", Num: id, Price: 5000}

}

func sessUserID(c echo.Context) int64 {
	sess, _ := session.Get("session", c)
	var userID int64
	if x, ok := sess.Values["user_id"]; ok {
		userID, _ = x.(int64)
	}
	return userID
}

func sessSetUserID(c echo.Context, id int64) {
	sess, _ := session.Get("session", c)
	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
	}
	sess.Values["user_id"] = id
	sess.Save(c.Request(), c.Response())
}

func sessDeleteUserID(c echo.Context) {
	sess, _ := session.Get("session", c)
	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
	}
	delete(sess.Values, "user_id")
	sess.Save(c.Request(), c.Response())
}

func sessAdministratorID(c echo.Context) int64 {
	sess, _ := session.Get("session", c)
	var administratorID int64
	if x, ok := sess.Values["administrator_id"]; ok {
		administratorID, _ = x.(int64)
	}
	return administratorID
}

func sessSetAdministratorID(c echo.Context, id int64) {
	sess, _ := session.Get("session", c)
	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
	}
	sess.Values["administrator_id"] = id
	sess.Save(c.Request(), c.Response())
}

func sessDeleteAdministratorID(c echo.Context) {
	sess, _ := session.Get("session", c)
	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
	}
	delete(sess.Values, "administrator_id")
	sess.Save(c.Request(), c.Response())
}

func loginRequired(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if _, err := getLoginUser(c); err != nil {
			return resError(c, "login_required", 401)
		}
		return next(c)
	}
}

func adminLoginRequired(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if _, err := getLoginAdministrator(c); err != nil {
			return resError(c, "admin_login_required", 401)
		}
		return next(c)
	}
}

func getLoginUser(c echo.Context) (*User, error) {
	userID := sessUserID(c)
	if userID == 0 {
		return nil, errors.New("not logged in")
	}
	var user User
	err := db.QueryRow("SELECT id, nickname FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Nickname)
	return &user, err
}

func getLoginAdministrator(c echo.Context) (*Administrator, error) {
	administratorID := sessAdministratorID(c)
	if administratorID == 0 {
		return nil, errors.New("not logged in")
	}
	var administrator Administrator
	err := db.QueryRow("SELECT id, nickname FROM administrators WHERE id = ?", administratorID).Scan(&administrator.ID, &administrator.Nickname)
	return &administrator, err
}

type Rvs map[int64]Reservation

var EMPTY_RVS = make(Rvs)

var (
	// rvss[eventID][sheetID]
	gRvss       = make(map[int64]map[int64]Reservation)
	gRvssLock   sync.Mutex
	gRvssRWLock sync.RWMutex
	gRvssLast   time.Time
)

func updateRvss() error {
	now := time.Now()
	gRvssLock.Lock()
	defer gRvssLock.Unlock()
	gRvssRWLock.Lock()
	defer gRvssRWLock.Unlock()

	if gRvssLast.After(now) {
		return nil
	}

	{
		rows2, err := db.Query("SELECT * FROM reservations WHERE canceled_at >= ? OR reserved_at >= ?", gRvssLast.Add(-2*time.Second).UTC().Format("2006-01-02 15:04:05.000000"), gRvssLast.Add(-2*time.Second).UTC().Format("2006-01-02 15:04:05.000000"))
		if err != nil {
			return err
		}
		defer rows2.Close()
		for rows2.Next() {
			var rv Reservation
			err := rows2.Scan(&rv.ID, &rv.EventID, &rv.SheetID, &rv.UserID, &rv.ReservedAt, &rv.CanceledAt, &rv.EventPrice)
			if err != nil {
				return err
			}

			if rv.CanceledAt.Unix() <= 0 {
				if _, ok := gRvss[rv.EventID]; !ok {
					gRvss[rv.EventID] = make(map[int64]Reservation)
				}
				gRvss[rv.EventID][rv.SheetID] = rv
			} else {
				rvs, ok := gRvss[rv.EventID]
				if !ok {
					continue
				}
				r, ok := rvs[rv.SheetID]
				if !ok {
					continue
				}
				if r.ID == rv.ID {
					delete(gRvss[rv.EventID], rv.SheetID)
				}
			}

		}
	}

	gRvssLast = now
	return nil
}

var gRvssLasts map[int64]time.Time

func updateRvssOnlyEvent(eid int64) error {
	now := time.Now()
	gRvssRWLock.Lock()
	defer gRvssRWLock.Unlock()
	if gRvssLast.After(now) {
		return nil
	}
	if t, ok := gRvssLasts[eid]; ok && t.After(now) {
		return nil
	}
	gRvssLasts[eid] = now

	{
		rows2, err := db.Query("SELECT * FROM reservations WHERE event_id = ? AND (canceled_at >= ? OR reserved_at >= ?)", eid, gRvssLast.Add(-2*time.Second).UTC().Format("2006-01-02 15:04:05.000000"), gRvssLast.Add(-2*time.Second).UTC().Format("2006-01-02 15:04:05.000000"))
		if err != nil {
			return err
		}
		defer rows2.Close()
		for rows2.Next() {
			var rv Reservation
			err := rows2.Scan(&rv.ID, &rv.EventID, &rv.SheetID, &rv.UserID, &rv.ReservedAt, &rv.CanceledAt, &rv.EventPrice)
			if err != nil {
				return err
			}

			if rv.CanceledAt.Unix() <= 0 {
				if _, ok := gRvss[rv.EventID]; !ok {
					gRvss[rv.EventID] = make(map[int64]Reservation)
				}
				gRvss[rv.EventID][rv.SheetID] = rv
			} else {
				rvs, ok := gRvss[rv.EventID]
				if !ok {
					continue
				}
				r, ok := rvs[rv.SheetID]
				if !ok {
					continue
				}
				if r.ID == rv.ID {
					delete(gRvss[rv.EventID], rv.SheetID)
				}
			}

		}
	}
	return nil
}

func getEvents(all bool) ([]*Event, error) {
	eg := errgroup.Group{}

	var events []*Event
	eg.Go(func() error {
		rows, err := db2.Query("SELECT * FROM events")
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var event Event
			if err := rows.Scan(&event.ID, &event.Title, &event.PublicFg, &event.ClosedFg, &event.Price); err != nil {
				return err
			}
			if !all && !event.PublicFg {
				continue
			}
			events = append(events, &event)
		}
		return nil
	})

	eg.Go(updateRvss)

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	gRvssRWLock.RLock()
	defer gRvssRWLock.RUnlock()
	for i, event := range events {
		rvs, ok := gRvss[event.ID]
		if !ok {
			rvs = EMPTY_RVS
		}

		event.Total = 1000
		event.Sheets = map[string]*Sheets{
			"S": &Sheets{
				Total: 50,
				Price: 5000 + event.Price,
			},
			"A": &Sheets{
				Total: 150,
				Price: 3000 + event.Price,
			},
			"B": &Sheets{
				Total: 300,
				Price: 1000 + event.Price,
			},
			"C": &Sheets{
				Total: 500,
				Price: 0 + event.Price,
			},
		}

		for i := 1; i <= 1000; i++ {
			j := int64(i)
			s := sheetIDtoSheet(j)
			sheet := &s

			reservation, ok := rvs[sheet.ID]
			if ok {
				sheet.Reserved = true
				sheet.ReservedAtUnix = reservation.ReservedAt.Unix()
			} else {
				event.Remains++
				event.Sheets[sheet.Rank].Remains++
			}
		}

		for k := range event.Sheets {
			event.Sheets[k].Detail = nil
		}
		events[i] = event
	}
	return events, nil
}

func getEvent(eventID, uid int64) (*Event, error) {
	eg := errgroup.Group{}
	eg.Go(func() error {
		defer measure.Start("updateRvssOnlyEvent").Stop()
		if err := updateRvssOnlyEvent(eventID); err != nil {
			return err
		}
		return nil
	})

	var event Event
	eg.Go(func() error {
		defer measure.Start("get event sql").Stop()
		if err := db2.QueryRow("SELECT * FROM events WHERE id = ?", eventID).Scan(&event.ID, &event.Title, &event.PublicFg, &event.ClosedFg, &event.Price); err != nil {
			return err
		}
		return nil
	})

	m := measure.Start("eg wait")
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	m.Stop()

	m = measure.Start("after wait")
	defer m.Stop()

	gRvssRWLock.RLock()
	defer gRvssRWLock.RUnlock()

	rvs := gRvss[eventID]

	event.Total = 1000
	event.Sheets = map[string]*Sheets{
		"S": &Sheets{
			Total: 50,
			Price: 5000 + event.Price,
		},
		"A": &Sheets{
			Total: 150,
			Price: 3000 + event.Price,
		},
		"B": &Sheets{
			Total: 300,
			Price: 1000 + event.Price,
		},
		"C": &Sheets{
			Total: 500,
			Price: 0 + event.Price,
		},
	}

	for i := 1; i <= 1000; i++ {
		j := int64(i)
		s := sheetIDtoSheet(j)
		sheet := &s

		reservation, ok := rvs[sheet.ID]
		if ok {
			sheet.Mine = reservation.UserID == uid
			sheet.Reserved = true
			sheet.ReservedAtUnix = reservation.ReservedAt.Unix()
		} else {
			event.Remains++
			event.Sheets[sheet.Rank].Remains++
		}

		event.Sheets[sheet.Rank].Detail = append(event.Sheets[sheet.Rank].Detail, sheet)
	}

	return &event, nil
}

func sanitizeEvent(e *Event) *Event {
	sanitized := *e
	sanitized.Price = 0
	sanitized.PublicFg = false
	sanitized.ClosedFg = false
	return &sanitized
}

func fillinUser(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if user, err := getLoginUser(c); err == nil {
			c.Set("user", user)
		}
		return next(c)
	}
}

func fillinAdministrator(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if administrator, err := getLoginAdministrator(c); err == nil {
			c.Set("administrator", administrator)
		}
		return next(c)
	}
}

func validateRank(rank string) bool {
	switch rank {
	case "A", "B", "C", "S":
		return true
	}
	return false
}

type Renderer struct {
}

func (r *Renderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	m := data.(echo.Map)
	switch name {
	case "index.tmpl":
		es, _ := json.Marshal(m["events"])
		us, _ := json.Marshal(m["user"])
		os, _ := m["origin"].(string)
		e := []byte(strings.Replace(string(es), "\"", "&#34;", -1))
		u := []byte(strings.Replace(string(us), "\"", "&#34;", -1))
		o := []byte(os)
		t := indexTmpl(e, u, o)
		for i := range t {
			w.Write(t[i])
		}
	case "admin.tmpl":
		es, _ := json.Marshal(m["events"])
		us, _ := json.Marshal(m["administrator"])
		os, _ := m["origin"].(string)
		e := []byte(strings.Replace(string(es), "\"", "&#34;", -1))
		u := []byte(strings.Replace(string(us), "\"", "&#34;", -1))
		o := []byte(os)
		t := adminTmpl(e, u, o)
		for i := range t {
			w.Write(t[i])
		}
	}
	return nil
}

var db, db2 *sql.DB

var (
	cachedEvents     []*Event
	cachedTime       time.Time
	cachedEventsLock sync.Mutex
)

func index(c echo.Context) error {
	f := func() error {
		var err error
		now := time.Now()
		cachedEventsLock.Lock()
		defer cachedEventsLock.Unlock()

		if cachedTime.After(now) {
			return nil
		}
		cachedTime = time.Now()
		cachedEvents, err = getEvents(false)
		for i, v := range cachedEvents {
			cachedEvents[i] = sanitizeEvent(v)
		}
		return err
	}
	if err := f(); err != nil {
		return err
	}
	return c.Render(200, "index.tmpl", echo.Map{
		"events": cachedEvents,
		"user":   c.Get("user"),
		"origin": c.Scheme() + "://" + c.Request().Host,
	})
}

func initialize3(c echo.Context) error {
	cmd := exec.Command("../../db/init.sh")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	return err
}

func initialize2(c echo.Context) error {
	gRvss = make(map[int64]map[int64]Reservation)
	gRvssLast = time.Time{}
	gRvssLasts = make(map[int64]time.Time, 100)
	if err := updateRvss(); err != nil {
		return err
	}
	return c.NoContent(204)
}

func initialize(c echo.Context) error {
	go exec.Command("curl", "http://172.16.21.1/initialize3").Run()

	cmd := exec.Command("../../db/init.sh")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		return nil
	}

	exec.Command("curl", "http://172.16.21.1/initialize2").Run()
	exec.Command("curl", "http://172.16.21.2:8080/initialize2").Run()
	exec.Command("curl", "http://172.16.21.3:8080/initialize2").Run()

	return c.NoContent(204)
}

func users(c echo.Context) error {
	var params struct {
		Nickname  string `json:"nickname"`
		LoginName string `json:"login_name"`
		Password  string `json:"password"`
	}
	c.Bind(&params)

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	var user User
	if err := tx.QueryRow("SELECT * FROM users WHERE login_name = ?", params.LoginName).Scan(&user.ID, &user.LoginName, &user.Nickname, &user.PassHash); err != sql.ErrNoRows {
		tx.Rollback()
		if err == nil {
			return resError(c, "duplicated", 409)
		}
		return err
	}

	res, err := tx.Exec("INSERT INTO users (login_name, pass_hash, nickname) VALUES (?, SHA2(?, 256), ?)", params.LoginName, params.Password, params.Nickname)
	if err != nil {
		tx.Rollback()
		return resError(c, "", 0)
	}
	userID, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return resError(c, "", 0)
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	return c.JSON(201, echo.Map{
		"id":       userID,
		"nickname": params.Nickname,
	})
}
func getUser(c echo.Context) error {
	eg := errgroup.Group{}

	var user User

	eg.Go(func() error {
		if err := db.QueryRow("SELECT id, nickname FROM users WHERE id = ?", c.Param("id")).Scan(&user.ID, &user.Nickname); err != nil {
			return err
		}
		loginUser, err := getLoginUser(c)
		if err != nil {
			return err
		}
		if user.ID != loginUser.ID {
			return resError(c, "forbidden", 403)
		}
		return nil
	})

	var recentReservations []Reservation
	eg.Go(func() error {
		rows, err := db.Query("SELECT * FROM reservations WHERE user_id = ? ORDER BY IF(canceled_at > '0000-00-00 00:00:00', canceled_at, reserved_at) DESC LIMIT 5", user.ID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var reservation Reservation
			if err := rows.Scan(&reservation.ID, &reservation.EventID, &reservation.SheetID, &reservation.UserID, &reservation.ReservedAt, &reservation.CanceledAt, &reservation.EventPrice); err != nil {
				return err
			}
			sheet := sheetIDtoSheet(reservation.SheetID)

			event, err := getEvent(reservation.EventID, -1)
			if err != nil {
				return err
			}
			price := event.Sheets[sheet.Rank].Price
			event.Sheets = nil
			event.Total = 0
			event.Remains = 0

			reservation.Event = event
			reservation.SheetRank = sheet.Rank
			reservation.SheetNum = sheet.Num
			reservation.Price = price
			reservation.ReservedAtUnix = reservation.ReservedAt.Unix()
			if reservation.CanceledAt.Unix() > 0 {
				reservation.CanceledAtUnix = reservation.CanceledAt.Unix()
			}
			recentReservations = append(recentReservations, reservation)
		}
		if recentReservations == nil {
			recentReservations = make([]Reservation, 0)
		}
		return nil
	})

	var totalPrice int
	eg.Go(func() error {
		if err := db.QueryRow("SELECT IFNULL(SUM(r.event_price + s.price), 0) FROM reservations r INNER JOIN sheets s ON s.id = r.sheet_id WHERE r.user_id = ? AND r.canceled_at = '0000-00-00 00:00:00'", user.ID).Scan(&totalPrice); err != nil {
			return err
		}
		return nil
	})

	var recentEvents []*Event
	eg.Go(func() error {
		rows, err := db.Query("SELECT event_id FROM reservations WHERE user_id = ? GROUP BY event_id ORDER BY MAX(IF(canceled_at > '0000-00-00 00:00:00', canceled_at, reserved_at)) DESC LIMIT 5", user.ID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var eventID int64
			if err := rows.Scan(&eventID); err != nil {
				return err
			}
			event, err := getEvent(eventID, -1)
			if err != nil {
				return err
			}
			for k := range event.Sheets {
				event.Sheets[k].Detail = nil
			}
			recentEvents = append(recentEvents, event)
		}
		if recentEvents == nil {
			recentEvents = make([]*Event, 0)
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		return err
	}

	return c.JSON(200, echo.Map{
		"id":                  user.ID,
		"nickname":            user.Nickname,
		"recent_reservations": recentReservations,
		"total_price":         totalPrice,
		"recent_events":       recentEvents,
	})
}

func login(c echo.Context) error {
	var params struct {
		LoginName string `json:"login_name"`
		Password  string `json:"password"`
	}
	c.Bind(&params)

	user := new(User)
	if err := db.QueryRow("SELECT * FROM users WHERE login_name = ?", params.LoginName).Scan(&user.ID, &user.LoginName, &user.Nickname, &user.PassHash); err != nil {
		if err == sql.ErrNoRows {
			return resError(c, "authentication_failed", 401)
		}
		return err
	}

	var passHash string
	if err := db.QueryRow("SELECT SHA2(?, 256)", params.Password).Scan(&passHash); err != nil {
		return err
	}
	if user.PassHash != passHash {
		return resError(c, "authentication_failed", 401)
	}

	sessSetUserID(c, user.ID)
	user, err := getLoginUser(c)
	if err != nil {
		return err
	}
	return c.JSON(200, user)
}

func logout(c echo.Context) error {
	sessDeleteUserID(c)
	return c.NoContent(204)
}

func getEventsReq(c echo.Context) error {
	events, err := getEvents(true)
	if err != nil {
		return err
	}
	for i, v := range events {
		events[i] = sanitizeEvent(v)
	}
	return c.JSON(200, events)
}

func getEventReq(c echo.Context) error {
	eventID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return resError(c, "not_found", 404)
	}

	loginUserID := sessUserID(c)
	if loginUserID == 0 {
		loginUserID = -1
	}

	event, err := getEvent(eventID, loginUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return resError(c, "not_found", 404)
		}
		return err
	} else if !event.PublicFg {
		return resError(c, "not_found", 404)
	}
	return c.JSON(200, sanitizeEvent(event))
}

func postReserve(c echo.Context) error {
	eventID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return resError(c, "not_found", 404)
	}
	var params struct {
		Rank string `json:"sheet_rank"`
	}
	c.Bind(&params)

	var user User
	user.ID = sessUserID(c)
	if user.ID == 0 {
		return errors.New("not logged in")
	}

	var eventPrice int64
	var publicFg bool
	if err := db2.QueryRow("SELECT price, public_fg FROM events WHERE id = ?", eventID).Scan(&eventPrice, &publicFg); err != nil {
		if err == sql.ErrNoRows {
			return resError(c, "invalid_event", 404)
		}
		return err
	}
	if !publicFg {
		return resError(c, "invalid_event", 404)
	}

	if !validateRank(params.Rank) {
		return resError(c, "invalid_rank", 400)
	}

	var sheet Sheet
	var reservationID int64

	errCh := make(chan error)
	ctx := c.Request().Context()

	go (func() {
		for i := 0; i < 20; i++ {
			if err := db.QueryRow("SELECT * FROM sheets WHERE id NOT IN (SELECT sheet_id FROM reservations WHERE event_id = ? AND canceled_at = '0000-00-00 00:00:00') AND `rank` = ? ORDER BY RAND() LIMIT 1", eventID, params.Rank).Scan(&sheet.ID, &sheet.Rank, &sheet.Num, &sheet.Price); err != nil {
				if err == sql.ErrNoRows {
					errCh <- resError(c, "sold_out", 409)
					return
				}
				errCh <- err
				return
			}

			tx, err := db.Begin()
			if err != nil {
				errCh <- err
				return
			}

			res, err := tx.ExecContext(ctx, "INSERT INTO reservations (event_id, sheet_id, user_id, reserved_at, event_price) VALUES (?, ?, ?, ?, ?)", eventID, sheet.ID, user.ID, time.Now().UTC().Format("2006-01-02 15:04:05.000000"), eventPrice)
			if err != nil {
				tx.Rollback()
				log.Println("re-try: rollback by", err)
				continue
			}
			reservationID, err = res.LastInsertId()
			if err != nil {
				tx.Rollback()
				log.Println("re-try: rollback by", err)
				continue
			}
			tx.Commit()
			errCh <- nil
			return
		}
		errCh <- resError(c, "max retry", 555)
		return
	})()

	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
		return c.JSON(202, echo.Map{
			"id":         reservationID,
			"sheet_rank": params.Rank,
			"sheet_num":  sheet.Num,
		})
	case <-ctx.Done():
		return ctx.Err()
	}

}

func deleteReserve(c echo.Context) error {
	eventID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return resError(c, "not_found", 404)
	}
	rank := c.Param("rank")
	n, _ := strconv.Atoi(c.Param("num"))
	num := int64(n)

	var user User
	user.ID = sessUserID(c)
	if user.ID == 0 {
		return errors.New("not logged in")
	}

	var publicFg bool
	if err := db2.QueryRow("SELECT public_fg FROM events WHERE id = ?", eventID).Scan(&publicFg); err != nil {
		if err == sql.ErrNoRows {
			return resError(c, "invalid_event", 404)
		}
		return err
	}
	if !publicFg {
		return resError(c, "invalid_event", 404)
	}

	if !validateRank(rank) {
		return resError(c, "invalid_rank", 404)
	}

	sheet := Sheet{
		Rank: rank,
		Num:  num,
	}
	switch rank {
	case "S":
		sheet.ID = num
	case "A":
		sheet.ID = num + 50
	case "B":
		sheet.ID = num + 200
	case "C":
		sheet.ID = num + 500
	default:
		return resError(c, "invalid_sheet", 404)
	}
	if sheetIDtoSheet(sheet.ID).Rank != rank || num < 1 || num > 1000 {
		return resError(c, "invalid_sheet", 404)
	}

	for i := 0; i < 20; i++ {
		res, err := db.Exec(
			"UPDATE reservations SET canceled_at = ? WHERE event_id = ? AND sheet_id = ? AND canceled_at = '0000-00-00 00:00:00' AND user_id = ?",
			time.Now().UTC().Format("2006-01-02 15:04:05.000000"), eventID, sheet.ID, user.ID,
		)
		if err != nil {
			log.Println(err)
			continue
		}

		n, err := res.RowsAffected()
		if err != nil {
			log.Println(err)
			continue
		}

		if n == 0 {
			var a int64
			if err := db.QueryRow(
				"SELECT 1 FROM reservations WHERE event_id = ? AND sheet_id = ? AND canceled_at = '0000-00-00 00:00:00'",
				eventID, sheet.ID,
			).Scan(&a); err != nil {
				if err == sql.ErrNoRows {
					return resError(c, "not_reserved", 400)
				}
				return resError(c, "not_permitted", 403)
			}
		}

		return c.NoContent(204)
	}

	return resError(c, "retry exceed", 555)
}

var (
	cachedAdminEvents     []*Event
	cachedAdminTime       time.Time
	cachedAdminEventsLock sync.Mutex
)

func getAdmin(c echo.Context) error {
	f := func() error {
		var err error
		now := time.Now()
		cachedAdminEventsLock.Lock()
		defer cachedAdminEventsLock.Unlock()
		if cachedAdminTime.After(now) {
			return nil
		}
		cachedAdminTime = time.Now()
		cachedAdminEvents, err = getEvents(true)
		return err
	}

	var events []*Event
	administrator := c.Get("administrator")
	if administrator != nil {
		if err := f(); err != nil {
			return err
		}
		events = cachedAdminEvents
	}
	return c.Render(200, "admin.tmpl", echo.Map{
		"events":        events,
		"administrator": administrator,
		"origin":        c.Scheme() + "://" + c.Request().Host,
	})
}

func adminLogin(c echo.Context) error {
	var params struct {
		LoginName string `json:"login_name"`
		Password  string `json:"password"`
	}
	c.Bind(&params)

	administrator := new(Administrator)
	if err := db.QueryRow("SELECT * FROM administrators WHERE login_name = ?", params.LoginName).Scan(&administrator.ID, &administrator.LoginName, &administrator.Nickname, &administrator.PassHash); err != nil {
		if err == sql.ErrNoRows {
			return resError(c, "authentication_failed", 401)
		}
		return err
	}

	var passHash string
	if err := db.QueryRow("SELECT SHA2(?, 256)", params.Password).Scan(&passHash); err != nil {
		return err
	}
	if administrator.PassHash != passHash {
		return resError(c, "authentication_failed", 401)
	}

	sessSetAdministratorID(c, administrator.ID)

	administrator, err := getLoginAdministrator(c)
	if err != nil {
		return err
	}
	return c.JSON(200, administrator)
}

func adminLogout(c echo.Context) error {
	sessDeleteAdministratorID(c)
	return c.NoContent(204)
}

func postAdminEvents(c echo.Context) error {
	var params struct {
		Title  string `json:"title"`
		Public bool   `json:"public"`
		Price  int    `json:"price"`
	}
	c.Bind(&params)

	res, err := db2.Exec("INSERT INTO events (id, title, public_fg, closed_fg, price) VALUES (RAND()*100000, ?, ?, 0, ?)", params.Title, params.Public, params.Price)
	if err != nil {
		return err
	}
	eventID, err := res.LastInsertId()
	if err != nil {
		return err
	}

	event, err := getEvent(eventID, -1)
	if err != nil {
		return err
	}
	return c.JSON(200, event)
}

func getAdminEvent(c echo.Context) error {
	eventID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return resError(c, "not_found", 404)
	}
	event, err := getEvent(eventID, -1)
	if err != nil {
		if err == sql.ErrNoRows {
			return resError(c, "not_found", 404)
		}
		return err
	}
	return c.JSON(200, event)
}

func editAdminEvent(c echo.Context) error {
	eventID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return resError(c, "not_found", 404)
	}

	var params struct {
		Public bool `json:"public"`
		Closed bool `json:"closed"`
	}
	c.Bind(&params)
	if params.Closed {
		params.Public = false
	}

	var event Event
	if err := db2.QueryRow("SELECT * FROM events WHERE id = ?", eventID).Scan(&event.ID, &event.Title, &event.PublicFg, &event.ClosedFg, &event.Price); err != nil {
		if err == sql.ErrNoRows {
			return resError(c, "not_found", 404)
		}
		return err
	}

	if event.ClosedFg {
		return resError(c, "cannot_edit_closed_event", 400)
	} else if event.PublicFg && params.Closed {
		return resError(c, "cannot_close_public_event", 400)
	}

	if _, err := db2.Exec("UPDATE events SET public_fg = ?, closed_fg = ? WHERE id = ?", params.Public, params.Closed, event.ID); err != nil {
		return err
	}

	event.PublicFg = params.Public
	event.ClosedFg = params.Closed

	c.JSON(200, event)
	return nil
}

func reportSales(c echo.Context) error {
	eventID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return resError(c, "not_found", 404)
	}

	event, err := getEvent(eventID, -1)
	if err != nil {
		return err
	}

	rows, err := db.Query("SELECT * FROM reservations WHERE event_id = ?", event.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	c.Response().Header().Set("Content-Type", `text/csv; charset=UTF-8`)
	c.Response().Header().Set("Content-Disposition", `attachment; filename="report.csv"`)
	body := c.Response()
	body.Write([]byte("reservation_id,event_id,rank,num,price,user_id,sold_at,canceled_at\n"))

	for rows.Next() {
		var reservation Reservation
		var sheet Sheet
		if err := rows.Scan(&reservation.ID, &reservation.EventID, &reservation.SheetID, &reservation.UserID, &reservation.ReservedAt, &reservation.CanceledAt, &reservation.EventPrice); err != nil {
			return err
		}
		sheet = sheetIDtoSheet(reservation.SheetID)
		report := Report{
			ReservationID: reservation.ID,
			EventID:       reservation.EventID,
			Rank:          sheet.Rank,
			Num:           sheet.Num,
			UserID:        reservation.UserID,
			SoldAt:        reservation.ReservedAt.Format("2006-01-02T15:04:05.000000Z"),
			Price:         reservation.EventPrice + sheet.Price,
		}
		if reservation.CanceledAt.Unix() > 0 {
			report.CanceledAt = reservation.CanceledAt.Format("2006-01-02T15:04:05.000000Z")
		}

		body.Write([]byte(fmt.Sprintf("%d,%d,%s,%d,%d,%d,%s,%s\n",
			report.ReservationID, report.EventID, report.Rank, report.Num, report.Price, report.UserID, report.SoldAt, report.CanceledAt)))
	}
	return nil
}

func reportSaleses(c echo.Context) error {
	rows, err := db.Query("select * from reservations")
	if err != nil {
		return err
	}
	defer rows.Close()

	c.Response().Header().Set("Content-Type", `text/csv; charset=UTF-8`)
	c.Response().Header().Set("Content-Disposition", `attachment; filename="report.csv"`)
	body := c.Response()
	body.Write([]byte("reservation_id,event_id,rank,num,price,user_id,sold_at,canceled_at\n"))

	for rows.Next() {
		var reservation Reservation
		var sheet Sheet
		if err := rows.Scan(&reservation.ID, &reservation.EventID, &reservation.SheetID, &reservation.UserID, &reservation.ReservedAt, &reservation.CanceledAt, &reservation.EventPrice); err != nil {
			return err
		}
		sheet = sheetIDtoSheet(reservation.SheetID)
		report := Report{
			ReservationID: reservation.ID,
			EventID:       reservation.EventID,
			Rank:          sheet.Rank,
			Num:           sheet.Num,
			UserID:        reservation.UserID,
			SoldAt:        reservation.ReservedAt.Format("2006-01-02T15:04:05.000000Z"),
			Price:         reservation.EventPrice + sheet.Price,
		}
		if reservation.CanceledAt.Unix() > 0 {
			report.CanceledAt = reservation.CanceledAt.Format("2006-01-02T15:04:05.000000Z")
		}

		body.Write([]byte(fmt.Sprintf("%d,%d,%s,%d,%d,%d,%s,%s\n",
			report.ReservationID, report.EventID, report.Rank, report.Num, report.Price, report.UserID, report.SoldAt, report.CanceledAt)))
	}
	return nil
}

func main() {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4",
		os.Getenv("DB_USER"), os.Getenv("DB_PASS"),
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"),
		os.Getenv("DB_DATABASE"),
	)

	dsn2 := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4",
		os.Getenv("DB_USER"), os.Getenv("DB_PASS"),
		"172.16.21.1", os.Getenv("DB_PORT"),
		os.Getenv("DB_DATABASE"),
	)

	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	db.SetMaxIdleConns(100)

	db2, err = sql.Open("mysql", dsn2)
	if err != nil {
		log.Fatal(err)
	}
	db2.SetMaxIdleConns(100)

	e := echo.New()
	e.Renderer = &Renderer{}
	e.Use(session.Middleware(sessions.NewCookieStore([]byte("secret"))))
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{Output: os.Stderr}))
	e.Static("/", "public")

	e.GET("/", index, fillinUser)
	e.GET("/initialize3", initialize3)
	e.GET("/initialize2", initialize2)
	e.GET("/initialize", initialize)
	e.POST("/api/users", users)
	e.GET("/api/users/:id", getUser, loginRequired)
	e.POST("/api/actions/login", login)
	e.POST("/api/actions/logout", logout, loginRequired)
	e.GET("/api/events", getEventsReq)
	e.GET("/api/events/:id", getEventReq)
	e.POST("/api/events/:id/actions/reserve", postReserve, loginRequired)
	e.DELETE("/api/events/:id/sheets/:rank/:num/reservation", deleteReserve, loginRequired)
	e.GET("/admin/", getAdmin, fillinAdministrator)
	e.POST("/admin/api/actions/login", adminLogin)
	e.POST("/admin/api/actions/logout", adminLogout, adminLoginRequired)
	e.POST("/admin/api/events", postAdminEvents, adminLoginRequired)
	e.GET("/admin/api/events/:id", getAdminEvent, adminLoginRequired)
	e.POST("/admin/api/events/:id/actions/edit", editAdminEvent, adminLoginRequired)
	e.GET("/admin/api/reports/events/:id/sales", reportSales, adminLoginRequired)
	e.GET("/admin/api/reports/sales", reportSaleses, adminLoginRequired)

	go (func() {
		for {
			updateRvss()
			time.Sleep(time.Second / 10)
		}
	})()

	if os.Getenv("DEBUG_ISUCON") == "" {
		echopprof.Wrap(e)

		e.GET("/debug/measure/reset", func(c echo.Context) error {
			measure.Reset()
			return nil
		})

		e.GET("/debug/measure/:sort", func(c echo.Context) error {
			stats := measure.GetStats()
			stats.SortDesc(c.Param("sort"))

			w := c.Response()
			for _, stat := range stats {
				fmt.Fprintf(w, "%s, %+v\n", stat.Key, stat)
			}

			return nil
		})

	} else {
		fmt.Println("debugging...")
	}

	n, err := os.Hostname()
	if n == "isu1" {
		os.Remove("/var/run/echo/echo.sock")
		l, err := net.Listen("unix", "/var/run/echo/echo.sock")
		if err != nil {
			log.Fatal(err)
		}
		e.Listener = l
	}

	e.Start(":8080")
}

type Report struct {
	ReservationID int64
	EventID       int64
	Rank          string
	Num           int64
	UserID        int64
	SoldAt        string
	CanceledAt    string
	Price         int64
}

func renderReportCSV(c echo.Context, reports []Report) error {
	c.Response().Header().Set("Content-Type", `text/csv; charset=UTF-8`)
	c.Response().Header().Set("Content-Disposition", `attachment; filename="report.csv"`)

	body := c.Response()
	body.Write([]byte("reservation_id,event_id,rank,num,price,user_id,sold_at,canceled_at\n"))
	for _, v := range reports {
		body.Write([]byte(fmt.Sprintf("%d,%d,%s,%d,%d,%d,%s,%s\n",
			v.ReservationID, v.EventID, v.Rank, v.Num, v.Price, v.UserID, v.SoldAt, v.CanceledAt)))
	}
	return nil
}

func resError(c echo.Context, e string, status int) error {
	if e == "" {
		e = "unknown"
	}
	if status < 100 {
		status = 500
	}
	return c.JSON(status, map[string]string{"error": e})
}
