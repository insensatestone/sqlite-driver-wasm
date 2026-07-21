# sqlite-driver-wasm
golang sql driver for sqlite wasm
## TO DO


## How to use
```golang

import(
	_ "github.com/insensatestone/sqlite-driver-wasm/pkg/driver/sqlite3"
)

func main() {
    db, err := sql.Open("sqlite3_wasm", path)
    if err != nil {
        return nil, err
    }
    defer db.close()
    ......
}
```
