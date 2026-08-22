package queue

// popLua atomically pops the highest-priority ready job and grants a lease.
// KEYS: ready, payload, lease
// ARGV: now_unix, ttl_seconds, worker_id
const popLua = `
local ready = KEYS[1]
local payload = KEYS[2]
local lease = KEYS[3]
local now = tonumber(redis.call('TIME')[1])
local ttl = tonumber(ARGV[2])
local items = redis.call('ZRANGE', ready, 0, 0)
if (not items) or (#items == 0) then
  return nil
end
local id = items[1]
local data = redis.call('HGET', payload, id)
if not data then
  redis.call('ZREM', ready, id)
  return nil
end
redis.call('ZREM', ready, id)
redis.call('ZADD', lease, now + ttl, id)
redis.call('HSET', payload, id .. ':owner', ARGV[3])
return {id, data}
`

// ackLua drops lease + payload after successful processing.
// KEYS: lease, payload, taskset
// ARGV: job_id
const ackLua = `
local lease = KEYS[1]
local payload = KEYS[2]
local taskset = KEYS[3]
local id = ARGV[1]
redis.call('ZREM', lease, id)
redis.call('HDEL', payload, id, id .. ':owner')
if taskset ~= '' then
  redis.call('SREM', taskset, id)
end
return 1
`

// reclaimLua moves expired leases back to the ready zset. Score keeps original
// priority by reading leftover payload if present; otherwise uses now.
// KEYS: ready, lease, payload
// ARGV: now_unix
const reclaimLua = `
local ready = KEYS[1]
local lease = KEYS[2]
local payload = KEYS[3]
local now = tonumber(redis.call('TIME')[1])
local expired = redis.call('ZRANGEBYSCORE', lease, '-inf', now)
local n = 0
for i, id in ipairs(expired) do
  redis.call('ZREM', lease, id)
  redis.call('HDEL', payload, id .. ':owner')
  if redis.call('HGET', payload, id) then
    redis.call('ZADD', ready, now, id)
    n = n + 1
  end
end
return n
`

// dropTaskLua removes every job belonging to a cancelled task.
// KEYS: ready, lease, payload, taskset
const dropTaskLua = `
local ready = KEYS[1]
local lease = KEYS[2]
local payload = KEYS[3]
local taskset = KEYS[4]
local ids = redis.call('SMEMBERS', taskset)
local n = 0
for i, id in ipairs(ids) do
  redis.call('ZREM', ready, id)
  redis.call('ZREM', lease, id)
  redis.call('HDEL', payload, id, id .. ':owner')
  n = n + 1
end
redis.call('DEL', taskset)
return n
`
