-- 修改 root 用户的认证插件为 mysql_native_password
ALTER USER 'root'@'%' IDENTIFIED WITH mysql_native_password BY 'root';

-- 如果你想为其他用户设置相同的插件，可以类似这样操作：
-- ALTER USER 'your_user'@'%' IDENTIFIED WITH mysql_native_password BY 'your_password';
