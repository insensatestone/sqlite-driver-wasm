# sqlite-driver-wasm
golang sql driver for sqlite wasm
## TO DO


## How to use
```golang

import(
	_ "gdream/internal/driver/sqlite"
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
