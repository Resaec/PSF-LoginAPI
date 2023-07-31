# PSF-LoginAPI
### A GO login API for the [PSF Project](https://www.psforever.net/) Game Launcher.

Allows the [PSF Launcher](https://github.com/psforever/GameLauncher) to perform a login gainst a PSF database.
A successful login will return a game login token that will be passed to the Planetside process.
Here it is used as authentication for **World Select** and **World** Servers.

### There are a few environment variables that are expected.

#### JWT token key
* JWT_KEY

#### Database credentials
* PG_HOST
* PG_PASS
* PG_PORT
* PG_USER
* PG_DB

#### GIN mode
* GIN_MODE

not required, but you should set it to `release`
