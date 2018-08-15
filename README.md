# static-forms (Beta)

[Official repo](https://git.shadow53.com/BluestNight/static-forms/)

[Official mirror](https://git.shadow53.com/BluestNight/nebula-forms/)

A simple Go http server that responds to POST requests from forms on static
sites like Hugo.

Development happens on Phabricator, just like the rest of my projects, but
GitHub is the official mirror to make for easier importing into Go projects -
Phabricator doesn't make for very nice import paths.

## Usage

Run the `static-forms` binary, optionally with a `-conf` flag to specify a
non-default configuration file. The `-help` flag shows the default
configuration file location.

The rest of the options are set in the configuration file, formatted in TOML.
(Documentation to come later with a versioned beta release)

The project can also be used as part of a separate Go server, allowing for
more configurable usage - such as using something other than TOML for
configuration files, modifying default configuration option names, handling
changed configuration files differently, and so on.

## Features

- Multiple handlers for the same path
- Namespaced handlers by domain - two domains can use the same path without
  triggering the other's submission handler
- Live reloading of configuration files
- Logging to stdout/stderr and log files
- Uses Golang templates for configurable output
- Supports the following handlers:
    - SMTP emails
