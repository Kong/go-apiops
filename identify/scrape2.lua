#!/usr/bin/env resty

--[[

This script is derived from the Kong 3.2.x.x CLI scripts. Specifically the
"config" command. It might need adapting for other Kong versions.

Since it is derived from Kong, it must also run on a Kong installation.

It will collect all DAO schema's, and export them as a JSON file (in the PWD).

Run this:

$ cd ./go-apiops
$ KONG_VERSION=stable-ee pongo shell
$ cd /kong-plugin/identify
$ ./scrape2.lua
$ ./list.lua

]]
require("kong.globalpatches")({cli = true})

math.randomseed() -- Generate PRNG seed

local DB = require "kong.db"
local log = require "kong.cmd.utils.log"
local kong_global = require "kong.global"
local declarative = require "kong.db.declarative"
local conf_loader = require "kong.conf_loader"
local writefile = require("pl.utils").writefile
local json_encode = require("cjson").encode

log.set_lvl(log.levels.info) -- one of: debug, verbose, info, warn, error, quiet

-- Load and setup Kong infra to collect schema's
local conf = assert(conf_loader())
package.path = conf.lua_package_path .. ";" .. package.path

_G.kong = kong_global.new()
kong_global.init_pdk(_G.kong, conf)
assert(declarative.new_config(conf, true))

_G.kong.db = assert(DB.new(conf))
assert(_G.kong.db.plugins:load_plugin_schemas(conf.loaded_plugins))



-- Collect the DAO schema's, include some cleanup
local schemas = {}


for name, dao in pairs(_G.kong.db.daos) do
  log.info("found DAO: " .. name)
  schemas[name] = dao.schema
end


local done = {} -- recursion-tracker
local function cleanup_schema(t)
  if done[t] then return end
  done[t] = true

  if t.reference and t.schema and type(t.schema) == "table" then
    -- remove nested sub-schema's, only retain the reference
    t.schema = nil
  end

  if t.entity_checks then
    -- for brevity drop the entity-checks since we do not use them
    t.entity_checks = nil
  end

  for key, val in pairs(t) do
    if type(val) == "function" then
      -- we cannot serialize functions, so replace with a string-placeholder
      t[key] = tostring(val)

    elseif type(val) == "table" then
      if key == "fields" then
        -- the fields part has both hash- and array-based entries. We drop the
        -- array part for brevity
        for i in ipairs(val) do
          val[i] = nil -- scrap array part, keep hash part
        end
      end

      -- since this is a table, recurse to walk the entire tree
      cleanup_schema(val)
    end
  end
end
cleanup_schema(schemas)

-- remove the plugin schema's (config schema's, which are non-DAO ones)
schemas.plugins.subschema_key = nil
schemas.plugins.subschema_error = nil
schemas.plugins.new_subschema = nil
schemas.plugins.subschemas = nil


-- write the output to a file
local filename = "./schemas.json"
assert(writefile(filename, json_encode(schemas)))
log.info("schemas written to: "..filename)

