#!/usr/bin/env lua

local getfiles = require("pl.dir").getfiles
local getallfiles = require("pl.dir").getallfiles
--local readfile = require("pl.utils").readfile
local insertvalues = require("pl.tablex").insertvalues
local yaml = require "lyaml"

-- make sure to pick up the local source tree when "require"'ing modules
package.path = "./?.lua;./?/init.lua;"..package.path

-- this goes to stderr, where "print" goes to stdout
local function log(...)
  io.stderr:write(...)
  io.stderr:write("\n")
end

-- local base_path = "./"
-- local filelist = {} do
--   local core = getfiles(base_path.."kong/db/schema/entities/", "*.lua") -- non-recursive
--   local plugins = getallfiles(base_path.."kong/plugins/", "daos.lua") -- recursive
--   local eeplugins = getallfiles(base_path.."plugins-ee/", "daos.lua") -- recursive

--   insertvalues(filelist, core)
--   insertvalues(filelist, plugins)
--   insertvalues(filelist, eeplugins)
-- end

-- local foreign_key_patt =
--     "type" ..     -- the type keyword
--     "%s*" ..      -- optional whitespace
--     "=" ..        -- assignment
--     "%s*" ..      -- optional whitespace
--     "[\"']" ..    -- opening double or single quotes
--     "foreign" ..  -- foreign key identifier
--     "[\"']"       -- closing double or single quotes


-- for x, filename in ipairs(filelist) do
--   local filetext = readfile(filename)
--   local s = 0
--   local e
--   while s do
--     s, e = filetext:find(foreign_key_patt, s+1)
--     if s then
--       -- find 2 {} wrappers around the match to grab the whole field
--       local field
--       for i = s, 1, -1 do
--         local s1, e1 = filetext:find("%b{}", i)
--         if s1 and s1<s and e1>e then
--           -- found 1st wrapper pair
--           for i = s1, 1, -1 do
--             local s2, e2 = filetext:find("%b{}", i)
--             if s2 and s2<s1 and e2>e1 then
--               -- found 2nd wrapper pair
--               field = filetext:sub(s2,e2)
--               break
--             end
--           end
--           if field then
--             break
--           end
--         end
--       end
--       if field then
--         if not filelist[filename] then
--           filelist[filename] = field
--         else
--           -- already exists, so we have multiple matches
--           if type(filelist[filename]) == "string" then
--             filelist[filename] = {
--               filelist[filename],
--               field,
--             }
--           else
--             table.insert(filelist[filename], field)
--           end
--         end
--       end
--     end
--   end
--   filelist[x] = nil
-- end

-- log("title: ", require("pl.pretty").write(filelist))





-- log "attempt 2"

-------------------------------------------------------------------------------
-- Create a list of files to parse from the file tree of the Kong codebase
-------------------------------------------------------------------------------

local base_path = "./"
local filelist = {} do
  local core = getfiles(base_path.."kong/db/schema/entities/", "*.lua") -- non-recursive
  local plugins = getallfiles(base_path.."kong/plugins/", "daos.lua") -- recursive
  local eeplugins = getallfiles(base_path.."plugins-ee/", "daos.lua") -- recursive

  insertvalues(filelist, core)
  insertvalues(filelist, plugins)
  insertvalues(filelist, eeplugins)
end

-------------------------------------------------------------------------------
-- Patch "require" and "ngx" to not really load anything when we load
-- the schema files.
-------------------------------------------------------------------------------

-- Replace typedefs module with a stub that returns only the typedef name,
-- instead of the actual typedefs
local typedefs = setmetatable({}, {
  __index = function(_, key)
    -- someone is looking up a typedef...
    local result = setmetatable({
      -- returning the base typedef
      typedef = key
    }, {
      __call = function(_, typedef_override_table)
        -- calling on the typedef, hence setting the overrides
        typedef_override_table.typedef = key
        return typedef_override_table
      end,
    })
    return result
  end,
})


-- Any other module required will return a "auto-table" that just behaves and
-- works, without erroring. Actual results we do not care about.
-- return an "auto-table"
local function exectable()
  return setmetatable({}, {
    __call = function(_)
      return exectable
    end,
    __index = function(_, _)
      return exectable()
    end,
    __mul = function()
      return 1
    end
  })
end


-- actually patch "require" and "ngx"
local old_ngx = _G.ngx
local old_require = require

if not old_ngx then
  _G.ngx = exectable()
end

function require(name)  -- luacheck: ignore
  if name == "kong.db.schema.typedefs" or
     name == "kong.db.schema" or
     name == "kong.enterprise_edition.db.typedefs" then
    --return typedefs
    return old_require(name)
  end
  return exectable()
end

-------------------------------------------------------------------------------
-- Load the actual files using all the stubs and patches above
-------------------------------------------------------------------------------
local daos = {}
for _, filename in ipairs(filelist) do
  local dao = loadfile(filename)()
  if not filename:match("/daos%.lua$") then
    -- regular entity; 1 dao (plugin daos.lua can contain multiple)
    -- wrap this single one into an array/table to level the playing filed
    dao = { dao }
  end

  for i, schema in ipairs(dao) do
    schema._source_file = filename
    if not schema.fields then
      log(("file '%s', DAO %d, has no 'fields' property, discarding..."):format(filename, i))
    else
      if daos[schema.name] ~= nil then
        error(("Duplicate DAO name; '%s', defined in '%s' and '%s'")
          :format(schema.name, daos[schema.name]._source_file, schema._source_file))
      end
      daos[schema.name] = schema
    end
  end
end

-------------------------------------------------------------------------------
-- Revert the patches
-------------------------------------------------------------------------------
require = old_require  -- luacheck: ignore
_G.ngx = old_ngx


-------------------------------------------------------------------------------
-- Walk the tree to find all foreign relationships
-------------------------------------------------------------------------------
-- print("title: ", require("pl.pretty").write(daos.routes))

local output = {}
for name, schema in pairs(daos) do
  local foreign = {}

  --print(name..":", require("pl.pretty").write(schema.primary_key))
  local dao = {
    TableName = name,
    PrimaryKey = schema.primary_key,
    Workspaceable = schema.workspaceable,
    -- SourceFile = schema._source_file,
  }
  output[#output+1] = dao

  for _, field_obj in ipairs(schema.fields) do
    local fieldname, definition = next(field_obj)
    if type(definition) == "table" and definition.type == "foreign" then
      foreign[#foreign+1] = {
        LocalField = fieldname,
        ForeignTable = definition.reference
      }
    end
  end
  if #foreign > 0 then -- export if non-empty only
    -- sort for deterministic output
    table.sort(foreign, function(a,b) return a.LocalField < b.LocalField end)
    dao.ForeignRelations = foreign
  end
end

-- sort for deterministic output
table.sort(output, function(a,b) return a.TableName < b.TableName end)


-------------------------------------------------------------------------------
-- Sanitize the raw table data
-------------------------------------------------------------------------------

-- These tables do not contain configuration data for deck files, so we can leave
-- them out.
local tables_to_skip = {
  "tags",                       -- tags are managed separately by decK
  "workspace_entity_counters",  -- this is generated data, not config
  "audit_requests",             -- runtime data
  "degraphql_routes",           -- TODO: check what this table is
  "legacy_files",               -- TODO: sounds deprecated...
  "graphql_ratelimiting_advanced_cost_decoration", -- TODO: check what this is, feels like runtime data
  "clustering_data_planes",     -- TODO: check, seems runtime data
  "keyring_meta",               -- TODO: check
  "ratelimiting_metrics",       -- runtime data
  "login_attempts",             -- runtime data
  "consumer_reset_secrets",     -- TODO: what is this?
}
for _, tablename in ipairs(tables_to_skip) do
  local idx
  for i, tabledata in ipairs(output) do
    if tabledata.TableName == tablename then
      idx = i
      break
    end
  end
  assert(idx, "table '"..tablename.."' (to be skipped) wasn't found")
  log("skipping table '"..tablename.."'")
  table.remove(output, idx)
end

-------------------------------------------------------------------------------
-- output a Go-lang codefile to stdout
-------------------------------------------------------------------------------
print([[
package identify

// Note: this file is generated

// strEntityStructure contains a yaml snippet holding the foreign relationships
// between Kong entities.
const strEntityStructure = `
]])
print(yaml.dump({output}))
print("`")

log"done!, file written to stdout"
