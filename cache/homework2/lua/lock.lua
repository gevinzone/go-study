local val = redis.call('get', KEYS[1])
if not val then
    -- key 不存在，新设置一个key
    return redis.call('set', KEYS[1], ARGV[1], 'PX', ARGV[2])
-- 上一次加锁成功
elseif val == ARGV[1] then
    -- 更新锁的过期时间
    redis.call('expire', KEYS[1], ARGV[2])
    return "OK"
-- 锁被别人持有
else
    return ""
end