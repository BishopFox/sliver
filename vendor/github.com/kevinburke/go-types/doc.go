/*
	Package types implements several types for dealing with REST API's and
	databases.

	PrefixUUID

	UUID's are very useful, but you often need to attach context to them; e.g.
	you cannot look at a UUID and know whether it points to a record in the
	accounts table or in the messages table. A PrefixUUID solves this problem,
	by embedding the additional useful information as part of the string.

		a, _ := types.NewPrefixUUID("account6740b44e-13b9-475d-af06-979627e0e0d6")
		fmt.Println(a.Prefix) // "account"
		fmt.Println(a.UUID.String()) "6740b44e-13b9-475d-af06-979627e0e0d6"
		fmt.Println(a.String()) "account6740b44e-13b9-475d-af06-979627e0e0d6"

	If we had to write this value to the database as a string it would take up
	43 bytes. Instead we use a UUID type and strip the prefix before saving it.

	The converse, Value(), only returns the UUID part by default, since this
	is the only thing the database knows about. You can also attach the prefix
	manually in your SQL, like so:

		SELECT 'job_' || id, created_at FROM jobs;

	This will get parsed as part of the Scan(), and then you don't need to
	do anything. Alternatively, you can attach the prefix in your model,
	immediately after the query.

		func Get(id types.PrefixUUID) *User {
			var uid types.PrefixUUID
			var email string
			db.Conn.QueryRow("SELECT * FROM users WHERE id = $1").Scan(&uid, &email)
			uid.Prefix = "usr"
			return &User{
				ID: uid,
				Email: email
			}
		}

	NullString

	A NullString is like the null string in `database/sql`, but can additionally
	be encoded/decoded via JSON.

		json.NewEncoder(os.Stdout).Encode(NullString{Valid: false})
		// Output: null
		json.NewEncoder(os.Stdout).Encode(NullString{Valid: true, String: "hello"})
		// Output: "hello"

	NullTime

	A NullTime behaves exactly like NullString, but the value is a time.Time.

		json.NewEncoder(os.Stdout).Encode(NullTime{Valid: false})
		// Output: null
		json.NewEncoder(os.Stdout).Encode(NullTime{Valid: true, Time: time.Now()})
		// Output: "2016-05-02T08:33:46.005852482-07:00"
*/
package types
