-- Platform assertions before application schema exists. Add domain assertions in future tasks.
DO $$
BEGIN
  IF current_database() <> 'postgres' THEN
    RAISE EXCEPTION 'Unexpected database';
  END IF;
  IF to_regclass('auth.users') IS NULL THEN
    RAISE EXCEPTION 'Supabase Auth platform schema is absent';
  END IF;
END
$$;
