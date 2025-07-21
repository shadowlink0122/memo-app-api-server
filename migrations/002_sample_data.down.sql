-- サンプルデータ削除（Down Migration）

-- サンプルメモ削除
DELETE FROM memos WHERE user_id IN (
    SELECT id FROM users WHERE username IN ('testuser', 'admin', 'demo')
);

-- サンプルユーザー削除
DELETE FROM users WHERE username IN ('testuser', 'admin', 'demo');
