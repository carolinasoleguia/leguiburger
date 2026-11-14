-- A private bucket alone does not override an existing broad storage policy.
-- A restrictive policy keeps browser roles out even if another permissive
-- policy allows access to other buckets. The backend service role bypasses RLS.
CREATE POLICY leguiburger_private_backups_api_only ON storage.objects
 AS RESTRICTIVE FOR ALL TO PUBLIC
 USING(bucket_id<>'business-backups') WITH CHECK(bucket_id<>'business-backups');
