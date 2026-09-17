--
-- PostgreSQL database cluster dump
--

\restrict Cf3kbEpCg4u4JMK9exQGV67aInakUhTbhqvxdZAR992yBC3uwLDowggMj4ETUDP

SET default_transaction_read_only = off;

SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;

--
-- Drop databases (except postgres and template1)
--

DROP DATABASE kada;
DROP DATABASE kada_ai;




--
-- Drop roles
--

DROP ROLE kada;
DROP ROLE postgres;


--
-- Roles
--

CREATE ROLE kada;
ALTER ROLE kada WITH NOSUPERUSER INHERIT NOCREATEROLE CREATEDB LOGIN NOREPLICATION NOBYPASSRLS PASSWORD 'SCRAM-SHA-256$4096:z1pv4BAFjORz3HW6qagorQ==$N5w4xXj/fRU6aKGbBApOMZuLgswefWyQQtHz6YqBeko=:qFg5yVZqca10vh2f7FR7GNZ2hhJHLjBVVcYwn/41Sp4=';
CREATE ROLE postgres;
ALTER ROLE postgres WITH SUPERUSER INHERIT CREATEROLE CREATEDB LOGIN REPLICATION BYPASSRLS PASSWORD 'SCRAM-SHA-256$4096:OT9FIL7lQUJlUZYzMbWE8g==$XBUY9vXzbnBlPTcVwYOlnrqhH0h06Xn98UELLcnEL4Y=:DYH+EtloAlWBDRGdb/ZqLt+iSKultnxVuLbl49VasrY=';

--
-- User Configurations
--








\unrestrict Cf3kbEpCg4u4JMK9exQGV67aInakUhTbhqvxdZAR992yBC3uwLDowggMj4ETUDP

--
-- Databases
--

--
-- Database "template1" dump
--

--
-- PostgreSQL database dump
--

\restrict NfW5hEiOipiDU2NdeRRfXnIQaLmo3iO6IZTFzRM9165Z3pRbknvZ17dkTtqOUtg

-- Dumped from database version 16.15 (Debian 16.15-1.pgdg13+2)
-- Dumped by pg_dump version 16.15 (Debian 16.15-1.pgdg13+2)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

UPDATE pg_catalog.pg_database SET datistemplate = false WHERE datname = 'template1';
DROP DATABASE template1;
--
-- Name: template1; Type: DATABASE; Schema: -; Owner: postgres
--

CREATE DATABASE template1 WITH TEMPLATE = template0 ENCODING = 'UTF8' LOCALE_PROVIDER = libc LOCALE = 'en_US.utf8';


ALTER DATABASE template1 OWNER TO postgres;

\unrestrict NfW5hEiOipiDU2NdeRRfXnIQaLmo3iO6IZTFzRM9165Z3pRbknvZ17dkTtqOUtg
\connect template1
\restrict NfW5hEiOipiDU2NdeRRfXnIQaLmo3iO6IZTFzRM9165Z3pRbknvZ17dkTtqOUtg

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: DATABASE template1; Type: COMMENT; Schema: -; Owner: postgres
--

COMMENT ON DATABASE template1 IS 'default template for new databases';


--
-- Name: template1; Type: DATABASE PROPERTIES; Schema: -; Owner: postgres
--

ALTER DATABASE template1 IS_TEMPLATE = true;


\unrestrict NfW5hEiOipiDU2NdeRRfXnIQaLmo3iO6IZTFzRM9165Z3pRbknvZ17dkTtqOUtg
\connect template1
\restrict NfW5hEiOipiDU2NdeRRfXnIQaLmo3iO6IZTFzRM9165Z3pRbknvZ17dkTtqOUtg

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: DATABASE template1; Type: ACL; Schema: -; Owner: postgres
--

REVOKE CONNECT,TEMPORARY ON DATABASE template1 FROM PUBLIC;
GRANT CONNECT ON DATABASE template1 TO PUBLIC;


--
-- PostgreSQL database dump complete
--

\unrestrict NfW5hEiOipiDU2NdeRRfXnIQaLmo3iO6IZTFzRM9165Z3pRbknvZ17dkTtqOUtg

--
-- Database "kada" dump
--

--
-- PostgreSQL database dump
--

\restrict WKDgcoJJYtGG5fUHZIeI6WZXEHhapLCEU9mNvyiXU0meTs1jmv3c8AVWkvnOJrn

-- Dumped from database version 16.15 (Debian 16.15-1.pgdg13+2)
-- Dumped by pg_dump version 16.15 (Debian 16.15-1.pgdg13+2)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: kada; Type: DATABASE; Schema: -; Owner: kada
--

CREATE DATABASE kada WITH TEMPLATE = template0 ENCODING = 'UTF8' LOCALE_PROVIDER = libc LOCALE = 'en_US.utf8';


ALTER DATABASE kada OWNER TO kada;

\unrestrict WKDgcoJJYtGG5fUHZIeI6WZXEHhapLCEU9mNvyiXU0meTs1jmv3c8AVWkvnOJrn
\connect kada
\restrict WKDgcoJJYtGG5fUHZIeI6WZXEHhapLCEU9mNvyiXU0meTs1jmv3c8AVWkvnOJrn

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: click_platform; Type: TYPE; Schema: public; Owner: kada
--

CREATE TYPE public.click_platform AS ENUM (
    'browser',
    'wechat',
    'qq',
    'weibo',
    'xiaohongshu',
    'sms',
    'unknown'
);


ALTER TYPE public.click_platform OWNER TO kada;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: api_tokens; Type: TABLE; Schema: public; Owner: kada
--

CREATE TABLE public.api_tokens (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    name character varying(100) NOT NULL,
    token_hash character varying(64) NOT NULL,
    last_used timestamp with time zone,
    created_at timestamp with time zone
);


ALTER TABLE public.api_tokens OWNER TO kada;

--
-- Name: api_tokens_id_seq; Type: SEQUENCE; Schema: public; Owner: kada
--

CREATE SEQUENCE public.api_tokens_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.api_tokens_id_seq OWNER TO kada;

--
-- Name: api_tokens_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: kada
--

ALTER SEQUENCE public.api_tokens_id_seq OWNED BY public.api_tokens.id;


--
-- Name: click_logs; Type: TABLE; Schema: public; Owner: kada
--

CREATE TABLE public.click_logs (
    id bigint NOT NULL,
    link_id bigint,
    ip character varying(45),
    user_agent text,
    platform public.click_platform DEFAULT 'unknown'::public.click_platform,
    referer text,
    country character varying(10),
    province character varying(50),
    city character varying(50),
    created_at timestamp with time zone,
    event_id character varying(64)
);


ALTER TABLE public.click_logs OWNER TO kada;

--
-- Name: click_logs_id_seq; Type: SEQUENCE; Schema: public; Owner: kada
--

CREATE SEQUENCE public.click_logs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.click_logs_id_seq OWNER TO kada;

--
-- Name: click_logs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: kada
--

ALTER SEQUENCE public.click_logs_id_seq OWNED BY public.click_logs.id;


--
-- Name: domains; Type: TABLE; Schema: public; Owner: kada
--

CREATE TABLE public.domains (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    name character varying(255) NOT NULL,
    verified boolean DEFAULT false,
    verified_at timestamp with time zone,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);


ALTER TABLE public.domains OWNER TO kada;

--
-- Name: domains_id_seq; Type: SEQUENCE; Schema: public; Owner: kada
--

CREATE SEQUENCE public.domains_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.domains_id_seq OWNER TO kada;

--
-- Name: domains_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: kada
--

ALTER SEQUENCE public.domains_id_seq OWNED BY public.domains.id;


--
-- Name: folders; Type: TABLE; Schema: public; Owner: kada
--

CREATE TABLE public.folders (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    name character varying(100) NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);


ALTER TABLE public.folders OWNER TO kada;

--
-- Name: folders_id_seq; Type: SEQUENCE; Schema: public; Owner: kada
--

CREATE SEQUENCE public.folders_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.folders_id_seq OWNER TO kada;

--
-- Name: folders_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: kada
--

ALTER SEQUENCE public.folders_id_seq OWNED BY public.folders.id;


--
-- Name: link_tags; Type: TABLE; Schema: public; Owner: kada
--

CREATE TABLE public.link_tags (
    link_id bigint NOT NULL,
    tag_id bigint NOT NULL
);


ALTER TABLE public.link_tags OWNER TO kada;

--
-- Name: links; Type: TABLE; Schema: public; Owner: kada
--

CREATE TABLE public.links (
    id bigint NOT NULL,
    short_code character varying(20) NOT NULL,
    original_url text NOT NULL,
    title character varying(500),
    description text,
    image_url text,
    domain character varying(255) DEFAULT 'kada.link'::character varying,
    password_hash character varying(255),
    expires_at timestamp with time zone,
    is_active boolean DEFAULT true,
    click_count bigint DEFAULT 0,
    user_id bigint,
    workspace_id bigint,
    folder_id bigint,
    utm_source character varying(255),
    utm_medium character varying(255),
    utm_campaign character varying(255),
    utm_term character varying(255),
    utm_content character varying(255),
    ios_url text,
    android_url text,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);


ALTER TABLE public.links OWNER TO kada;

--
-- Name: links_id_seq; Type: SEQUENCE; Schema: public; Owner: kada
--

CREATE SEQUENCE public.links_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.links_id_seq OWNER TO kada;

--
-- Name: links_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: kada
--

ALTER SEQUENCE public.links_id_seq OWNED BY public.links.id;


--
-- Name: sms_codes; Type: TABLE; Schema: public; Owner: kada
--

CREATE TABLE public.sms_codes (
    id bigint NOT NULL,
    phone character varying(20) NOT NULL,
    code_hash character varying(64),
    ip character varying(45),
    used boolean DEFAULT false,
    attempts bigint DEFAULT 0 NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone
);


ALTER TABLE public.sms_codes OWNER TO kada;

--
-- Name: sms_codes_id_seq; Type: SEQUENCE; Schema: public; Owner: kada
--

CREATE SEQUENCE public.sms_codes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.sms_codes_id_seq OWNER TO kada;

--
-- Name: sms_codes_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: kada
--

ALTER SEQUENCE public.sms_codes_id_seq OWNED BY public.sms_codes.id;


--
-- Name: tags; Type: TABLE; Schema: public; Owner: kada
--

CREATE TABLE public.tags (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    name character varying(50) NOT NULL,
    color character varying(7) DEFAULT '#6366F1'::character varying,
    created_at timestamp with time zone
);


ALTER TABLE public.tags OWNER TO kada;

--
-- Name: tags_id_seq; Type: SEQUENCE; Schema: public; Owner: kada
--

CREATE SEQUENCE public.tags_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.tags_id_seq OWNER TO kada;

--
-- Name: tags_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: kada
--

ALTER SEQUENCE public.tags_id_seq OWNED BY public.tags.id;


--
-- Name: users; Type: TABLE; Schema: public; Owner: kada
--

CREATE TABLE public.users (
    id bigint NOT NULL,
    phone character varying(20),
    email character varying(255),
    wechat_openid character varying(128),
    wechat_unionid character varying(128),
    name character varying(100),
    avatar text,
    password_hash character varying(255),
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    last_login_at timestamp with time zone,
    last_login_ip character varying(45)
);


ALTER TABLE public.users OWNER TO kada;

--
-- Name: users_id_seq; Type: SEQUENCE; Schema: public; Owner: kada
--

CREATE SEQUENCE public.users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.users_id_seq OWNER TO kada;

--
-- Name: users_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: kada
--

ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;


--
-- Name: utm_templates; Type: TABLE; Schema: public; Owner: kada
--

CREATE TABLE public.utm_templates (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    name character varying(255) NOT NULL,
    utm_source character varying(255),
    utm_medium character varying(255),
    utm_campaign character varying(255),
    utm_term character varying(255),
    utm_content character varying(255),
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);


ALTER TABLE public.utm_templates OWNER TO kada;

--
-- Name: utm_templates_id_seq; Type: SEQUENCE; Schema: public; Owner: kada
--

CREATE SEQUENCE public.utm_templates_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.utm_templates_id_seq OWNER TO kada;

--
-- Name: utm_templates_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: kada
--

ALTER SEQUENCE public.utm_templates_id_seq OWNED BY public.utm_templates.id;


--
-- Name: workspaces; Type: TABLE; Schema: public; Owner: kada
--

CREATE TABLE public.workspaces (
    id bigint NOT NULL,
    name character varying(100) NOT NULL,
    slug character varying(50) NOT NULL,
    user_id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);


ALTER TABLE public.workspaces OWNER TO kada;

--
-- Name: workspaces_id_seq; Type: SEQUENCE; Schema: public; Owner: kada
--

CREATE SEQUENCE public.workspaces_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.workspaces_id_seq OWNER TO kada;

--
-- Name: workspaces_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: kada
--

ALTER SEQUENCE public.workspaces_id_seq OWNED BY public.workspaces.id;


--
-- Name: api_tokens id; Type: DEFAULT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.api_tokens ALTER COLUMN id SET DEFAULT nextval('public.api_tokens_id_seq'::regclass);


--
-- Name: click_logs id; Type: DEFAULT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.click_logs ALTER COLUMN id SET DEFAULT nextval('public.click_logs_id_seq'::regclass);


--
-- Name: domains id; Type: DEFAULT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.domains ALTER COLUMN id SET DEFAULT nextval('public.domains_id_seq'::regclass);


--
-- Name: folders id; Type: DEFAULT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.folders ALTER COLUMN id SET DEFAULT nextval('public.folders_id_seq'::regclass);


--
-- Name: links id; Type: DEFAULT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.links ALTER COLUMN id SET DEFAULT nextval('public.links_id_seq'::regclass);


--
-- Name: sms_codes id; Type: DEFAULT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.sms_codes ALTER COLUMN id SET DEFAULT nextval('public.sms_codes_id_seq'::regclass);


--
-- Name: tags id; Type: DEFAULT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.tags ALTER COLUMN id SET DEFAULT nextval('public.tags_id_seq'::regclass);


--
-- Name: users id; Type: DEFAULT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.users ALTER COLUMN id SET DEFAULT nextval('public.users_id_seq'::regclass);


--
-- Name: utm_templates id; Type: DEFAULT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.utm_templates ALTER COLUMN id SET DEFAULT nextval('public.utm_templates_id_seq'::regclass);


--
-- Name: workspaces id; Type: DEFAULT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.workspaces ALTER COLUMN id SET DEFAULT nextval('public.workspaces_id_seq'::regclass);


--
-- Data for Name: api_tokens; Type: TABLE DATA; Schema: public; Owner: kada
--

COPY public.api_tokens (id, user_id, name, token_hash, last_used, created_at) FROM stdin;
\.


--
-- Data for Name: click_logs; Type: TABLE DATA; Schema: public; Owner: kada
--

COPY public.click_logs (id, link_id, ip, user_agent, platform, referer, country, province, city, created_at, event_id) FROM stdin;
\.


--
-- Data for Name: domains; Type: TABLE DATA; Schema: public; Owner: kada
--

COPY public.domains (id, user_id, name, verified, verified_at, created_at, updated_at) FROM stdin;
\.


--
-- Data for Name: folders; Type: TABLE DATA; Schema: public; Owner: kada
--

COPY public.folders (id, user_id, name, created_at, updated_at) FROM stdin;
\.


--
-- Data for Name: link_tags; Type: TABLE DATA; Schema: public; Owner: kada
--

COPY public.link_tags (link_id, tag_id) FROM stdin;
\.


--
-- Data for Name: links; Type: TABLE DATA; Schema: public; Owner: kada
--

COPY public.links (id, short_code, original_url, title, description, image_url, domain, password_hash, expires_at, is_active, click_count, user_id, workspace_id, folder_id, utm_source, utm_medium, utm_campaign, utm_term, utm_content, ios_url, android_url, created_at, updated_at) FROM stdin;
\.


--
-- Data for Name: sms_codes; Type: TABLE DATA; Schema: public; Owner: kada
--

COPY public.sms_codes (id, phone, code_hash, ip, used, attempts, expires_at, created_at) FROM stdin;
\.


--
-- Data for Name: tags; Type: TABLE DATA; Schema: public; Owner: kada
--

COPY public.tags (id, user_id, name, color, created_at) FROM stdin;
\.


--
-- Data for Name: users; Type: TABLE DATA; Schema: public; Owner: kada
--

COPY public.users (id, phone, email, wechat_openid, wechat_unionid, name, avatar, password_hash, created_at, updated_at, last_login_at, last_login_ip) FROM stdin;
1	\N	test@test.com	\N	\N	t	\N	$2a$10$P9EPSONH/x8Cl3q0LAEiluCt28DzLDdaSTlELDwF5nif51612Qsf6	2026-09-17 03:11:22.649898+00	2026-09-17 03:11:22.649898+00	\N	\N
2	\N	2890008225@qq.com	\N	\N	杞梓	\N	$2a$10$BRDkvT3uPXIKUB9eMO3H/.2mz9afeVx55ulRvZO7w7bwSVswysPsy	2026-09-17 06:25:41.293086+00	2026-09-17 06:25:41.293086+00	\N	\N
\.


--
-- Data for Name: utm_templates; Type: TABLE DATA; Schema: public; Owner: kada
--

COPY public.utm_templates (id, user_id, name, utm_source, utm_medium, utm_campaign, utm_term, utm_content, created_at, updated_at) FROM stdin;
\.


--
-- Data for Name: workspaces; Type: TABLE DATA; Schema: public; Owner: kada
--

COPY public.workspaces (id, name, slug, user_id, created_at, updated_at) FROM stdin;
\.


--
-- Name: api_tokens_id_seq; Type: SEQUENCE SET; Schema: public; Owner: kada
--

SELECT pg_catalog.setval('public.api_tokens_id_seq', 1, false);


--
-- Name: click_logs_id_seq; Type: SEQUENCE SET; Schema: public; Owner: kada
--

SELECT pg_catalog.setval('public.click_logs_id_seq', 1, false);


--
-- Name: domains_id_seq; Type: SEQUENCE SET; Schema: public; Owner: kada
--

SELECT pg_catalog.setval('public.domains_id_seq', 1, false);


--
-- Name: folders_id_seq; Type: SEQUENCE SET; Schema: public; Owner: kada
--

SELECT pg_catalog.setval('public.folders_id_seq', 1, false);


--
-- Name: links_id_seq; Type: SEQUENCE SET; Schema: public; Owner: kada
--

SELECT pg_catalog.setval('public.links_id_seq', 1, false);


--
-- Name: sms_codes_id_seq; Type: SEQUENCE SET; Schema: public; Owner: kada
--

SELECT pg_catalog.setval('public.sms_codes_id_seq', 1, false);


--
-- Name: tags_id_seq; Type: SEQUENCE SET; Schema: public; Owner: kada
--

SELECT pg_catalog.setval('public.tags_id_seq', 1, false);


--
-- Name: users_id_seq; Type: SEQUENCE SET; Schema: public; Owner: kada
--

SELECT pg_catalog.setval('public.users_id_seq', 2, true);


--
-- Name: utm_templates_id_seq; Type: SEQUENCE SET; Schema: public; Owner: kada
--

SELECT pg_catalog.setval('public.utm_templates_id_seq', 1, false);


--
-- Name: workspaces_id_seq; Type: SEQUENCE SET; Schema: public; Owner: kada
--

SELECT pg_catalog.setval('public.workspaces_id_seq', 1, false);


--
-- Name: api_tokens api_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.api_tokens
    ADD CONSTRAINT api_tokens_pkey PRIMARY KEY (id);


--
-- Name: click_logs click_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.click_logs
    ADD CONSTRAINT click_logs_pkey PRIMARY KEY (id);


--
-- Name: domains domains_pkey; Type: CONSTRAINT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.domains
    ADD CONSTRAINT domains_pkey PRIMARY KEY (id);


--
-- Name: folders folders_pkey; Type: CONSTRAINT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.folders
    ADD CONSTRAINT folders_pkey PRIMARY KEY (id);


--
-- Name: link_tags link_tags_pkey; Type: CONSTRAINT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.link_tags
    ADD CONSTRAINT link_tags_pkey PRIMARY KEY (link_id, tag_id);


--
-- Name: links links_pkey; Type: CONSTRAINT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.links
    ADD CONSTRAINT links_pkey PRIMARY KEY (id);


--
-- Name: sms_codes sms_codes_pkey; Type: CONSTRAINT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.sms_codes
    ADD CONSTRAINT sms_codes_pkey PRIMARY KEY (id);


--
-- Name: tags tags_pkey; Type: CONSTRAINT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.tags
    ADD CONSTRAINT tags_pkey PRIMARY KEY (id);


--
-- Name: users uni_users_email; Type: CONSTRAINT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT uni_users_email UNIQUE (email);


--
-- Name: users uni_users_phone; Type: CONSTRAINT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT uni_users_phone UNIQUE (phone);


--
-- Name: users uni_users_wechat_openid; Type: CONSTRAINT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT uni_users_wechat_openid UNIQUE (wechat_openid);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: utm_templates utm_templates_pkey; Type: CONSTRAINT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.utm_templates
    ADD CONSTRAINT utm_templates_pkey PRIMARY KEY (id);


--
-- Name: workspaces workspaces_pkey; Type: CONSTRAINT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.workspaces
    ADD CONSTRAINT workspaces_pkey PRIMARY KEY (id);


--
-- Name: idx_api_tokens_token_hash; Type: INDEX; Schema: public; Owner: kada
--

CREATE UNIQUE INDEX idx_api_tokens_token_hash ON public.api_tokens USING btree (token_hash);


--
-- Name: idx_api_tokens_user_id; Type: INDEX; Schema: public; Owner: kada
--

CREATE INDEX idx_api_tokens_user_id ON public.api_tokens USING btree (user_id);


--
-- Name: idx_click_logs_created_at; Type: INDEX; Schema: public; Owner: kada
--

CREATE INDEX idx_click_logs_created_at ON public.click_logs USING btree (created_at);


--
-- Name: idx_click_logs_event_id; Type: INDEX; Schema: public; Owner: kada
--

CREATE UNIQUE INDEX idx_click_logs_event_id ON public.click_logs USING btree (event_id);


--
-- Name: idx_click_logs_link_id; Type: INDEX; Schema: public; Owner: kada
--

CREATE INDEX idx_click_logs_link_id ON public.click_logs USING btree (link_id);


--
-- Name: idx_click_logs_platform; Type: INDEX; Schema: public; Owner: kada
--

CREATE INDEX idx_click_logs_platform ON public.click_logs USING btree (platform);


--
-- Name: idx_domains_user_id; Type: INDEX; Schema: public; Owner: kada
--

CREATE INDEX idx_domains_user_id ON public.domains USING btree (user_id);


--
-- Name: idx_domains_user_name; Type: INDEX; Schema: public; Owner: kada
--

CREATE UNIQUE INDEX idx_domains_user_name ON public.domains USING btree (user_id, name);


--
-- Name: idx_folders_user_id; Type: INDEX; Schema: public; Owner: kada
--

CREATE INDEX idx_folders_user_id ON public.folders USING btree (user_id);


--
-- Name: idx_links_created_at; Type: INDEX; Schema: public; Owner: kada
--

CREATE INDEX idx_links_created_at ON public.links USING btree (created_at);


--
-- Name: idx_links_folder_id; Type: INDEX; Schema: public; Owner: kada
--

CREATE INDEX idx_links_folder_id ON public.links USING btree (folder_id);


--
-- Name: idx_links_short_code; Type: INDEX; Schema: public; Owner: kada
--

CREATE UNIQUE INDEX idx_links_short_code ON public.links USING btree (short_code);


--
-- Name: idx_links_user_id; Type: INDEX; Schema: public; Owner: kada
--

CREATE INDEX idx_links_user_id ON public.links USING btree (user_id);


--
-- Name: idx_links_workspace_id; Type: INDEX; Schema: public; Owner: kada
--

CREATE INDEX idx_links_workspace_id ON public.links USING btree (workspace_id);


--
-- Name: idx_sms_codes_code_hash; Type: INDEX; Schema: public; Owner: kada
--

CREATE INDEX idx_sms_codes_code_hash ON public.sms_codes USING btree (code_hash);


--
-- Name: idx_sms_codes_phone; Type: INDEX; Schema: public; Owner: kada
--

CREATE INDEX idx_sms_codes_phone ON public.sms_codes USING btree (phone, created_at);


--
-- Name: idx_tags_user_id; Type: INDEX; Schema: public; Owner: kada
--

CREATE INDEX idx_tags_user_id ON public.tags USING btree (user_id);


--
-- Name: idx_users_email; Type: INDEX; Schema: public; Owner: kada
--

CREATE UNIQUE INDEX idx_users_email ON public.users USING btree (email);


--
-- Name: idx_users_phone; Type: INDEX; Schema: public; Owner: kada
--

CREATE UNIQUE INDEX idx_users_phone ON public.users USING btree (phone);


--
-- Name: idx_users_wechat_openid; Type: INDEX; Schema: public; Owner: kada
--

CREATE UNIQUE INDEX idx_users_wechat_openid ON public.users USING btree (wechat_openid);


--
-- Name: idx_utm_templates_user_id; Type: INDEX; Schema: public; Owner: kada
--

CREATE INDEX idx_utm_templates_user_id ON public.utm_templates USING btree (user_id);


--
-- Name: idx_workspaces_slug; Type: INDEX; Schema: public; Owner: kada
--

CREATE UNIQUE INDEX idx_workspaces_slug ON public.workspaces USING btree (slug);


--
-- Name: idx_workspaces_user_id; Type: INDEX; Schema: public; Owner: kada
--

CREATE INDEX idx_workspaces_user_id ON public.workspaces USING btree (user_id);


--
-- Name: api_tokens fk_api_tokens_user; Type: FK CONSTRAINT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.api_tokens
    ADD CONSTRAINT fk_api_tokens_user FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: click_logs fk_click_logs_link; Type: FK CONSTRAINT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.click_logs
    ADD CONSTRAINT fk_click_logs_link FOREIGN KEY (link_id) REFERENCES public.links(id) ON DELETE CASCADE;


--
-- Name: domains fk_domains_user; Type: FK CONSTRAINT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.domains
    ADD CONSTRAINT fk_domains_user FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: folders fk_folders_user; Type: FK CONSTRAINT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.folders
    ADD CONSTRAINT fk_folders_user FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: link_tags fk_link_tags_link; Type: FK CONSTRAINT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.link_tags
    ADD CONSTRAINT fk_link_tags_link FOREIGN KEY (link_id) REFERENCES public.links(id) ON DELETE CASCADE;


--
-- Name: link_tags fk_link_tags_tag; Type: FK CONSTRAINT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.link_tags
    ADD CONSTRAINT fk_link_tags_tag FOREIGN KEY (tag_id) REFERENCES public.tags(id) ON DELETE CASCADE;


--
-- Name: links fk_links_folder; Type: FK CONSTRAINT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.links
    ADD CONSTRAINT fk_links_folder FOREIGN KEY (folder_id) REFERENCES public.folders(id) ON DELETE SET NULL;


--
-- Name: links fk_links_user; Type: FK CONSTRAINT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.links
    ADD CONSTRAINT fk_links_user FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: links fk_links_workspace; Type: FK CONSTRAINT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.links
    ADD CONSTRAINT fk_links_workspace FOREIGN KEY (workspace_id) REFERENCES public.workspaces(id) ON DELETE SET NULL;


--
-- Name: tags fk_tags_user; Type: FK CONSTRAINT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.tags
    ADD CONSTRAINT fk_tags_user FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: utm_templates fk_utm_templates_user; Type: FK CONSTRAINT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.utm_templates
    ADD CONSTRAINT fk_utm_templates_user FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: workspaces fk_workspaces_user; Type: FK CONSTRAINT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.workspaces
    ADD CONSTRAINT fk_workspaces_user FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- PostgreSQL database dump complete
--

\unrestrict WKDgcoJJYtGG5fUHZIeI6WZXEHhapLCEU9mNvyiXU0meTs1jmv3c8AVWkvnOJrn

--
-- Database "kada_ai" dump
--

--
-- PostgreSQL database dump
--

\restrict eFzbyQgLqPHyNPS4G19vbLXThKUBdr2J15hykuIj7mqZImvV2mbZkBG9uXVsqrQ

-- Dumped from database version 16.15 (Debian 16.15-1.pgdg13+2)
-- Dumped by pg_dump version 16.15 (Debian 16.15-1.pgdg13+2)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: kada_ai; Type: DATABASE; Schema: -; Owner: postgres
--

CREATE DATABASE kada_ai WITH TEMPLATE = template0 ENCODING = 'UTF8' LOCALE_PROVIDER = libc LOCALE = 'en_US.utf8';


ALTER DATABASE kada_ai OWNER TO postgres;

\unrestrict eFzbyQgLqPHyNPS4G19vbLXThKUBdr2J15hykuIj7mqZImvV2mbZkBG9uXVsqrQ
\connect kada_ai
\restrict eFzbyQgLqPHyNPS4G19vbLXThKUBdr2J15hykuIj7mqZImvV2mbZkBG9uXVsqrQ

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: ai_conversations; Type: TABLE; Schema: public; Owner: kada
--

CREATE TABLE public.ai_conversations (
    id uuid NOT NULL,
    user_id character varying NOT NULL,
    created_at timestamp with time zone NOT NULL
);


ALTER TABLE public.ai_conversations OWNER TO kada;

--
-- Name: ai_messages; Type: TABLE; Schema: public; Owner: kada
--

CREATE TABLE public.ai_messages (
    id integer NOT NULL,
    conversation_id uuid NOT NULL,
    role character varying NOT NULL,
    content character varying NOT NULL,
    created_at timestamp with time zone NOT NULL
);


ALTER TABLE public.ai_messages OWNER TO kada;

--
-- Name: ai_messages_id_seq; Type: SEQUENCE; Schema: public; Owner: kada
--

CREATE SEQUENCE public.ai_messages_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.ai_messages_id_seq OWNER TO kada;

--
-- Name: ai_messages_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: kada
--

ALTER SEQUENCE public.ai_messages_id_seq OWNED BY public.ai_messages.id;


--
-- Name: ai_messages id; Type: DEFAULT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.ai_messages ALTER COLUMN id SET DEFAULT nextval('public.ai_messages_id_seq'::regclass);


--
-- Data for Name: ai_conversations; Type: TABLE DATA; Schema: public; Owner: kada
--

COPY public.ai_conversations (id, user_id, created_at) FROM stdin;
bcd8d110-0f98-4050-81ff-effde7720820	demo-user	2026-09-16 08:36:33.245367+00
0781553f-2af3-4e63-af60-7225f98f0f8b	demo-user	2026-09-16 08:37:13.356894+00
174bc880-2663-4065-b34d-72fe35abe575	demo-user	2026-09-16 08:38:30.272647+00
608f8e79-a211-4206-ace6-7cb9b26fbf75	1	2026-09-16 10:46:07.446108+00
2f740a46-b199-43d6-9f88-0a28df8d93eb	1	2026-09-16 10:50:26.376837+00
38e3d66e-d629-4d9b-bd66-6da9ae02b5e9	demo-user	2026-09-16 12:50:05.967987+00
ed8674ad-99e9-463f-a9ce-720145148a0d	1	2026-09-16 15:31:59.788085+00
4673e2ae-b890-41a9-90a8-60aadcabeb05	1	2026-09-16 15:56:54.503819+00
3817676e-8338-49c6-9102-ad17951fb56a	2	2026-09-17 06:26:02.158599+00
06f6b042-ad13-4873-8bfc-23f32f6b57c3	2	2026-09-17 10:50:36.850376+00
\.


--
-- Data for Name: ai_messages; Type: TABLE DATA; Schema: public; Owner: kada
--

COPY public.ai_messages (id, conversation_id, role, content, created_at) FROM stdin;
1	174bc880-2663-4065-b34d-72fe35abe575	user	你好	2026-09-16 08:38:30.321889+00
2	174bc880-2663-4065-b34d-72fe35abe575	assistant	你好！很高兴见到你 😊 有什么我可以帮忙的吗？无论是问题解答、写作、翻译、编程，还是只是想聊聊天，都可以告诉我。	2026-09-16 08:38:32.519016+00
3	608f8e79-a211-4206-ace6-7cb9b26fbf75	user	你是	2026-09-16 10:46:07.501036+00
4	608f8e79-a211-4206-ace6-7cb9b26fbf75	assistant	我是 AI 助手，可以帮你解答问题、写作、翻译、编程等。你想问我什么？	2026-09-16 10:46:10.773253+00
5	608f8e79-a211-4206-ace6-7cb9b26fbf75	user	给我一段长约500字的内容吧我向来测试一下你的流式输出功能	2026-09-16 10:47:08.969012+00
6	2f740a46-b199-43d6-9f88-0a28df8d93eb	user	给我一段长约500字的内容吧我向来测试一下你的流式输出功能	2026-09-16 10:50:26.41679+00
7	2f740a46-b199-43d6-9f88-0a28df8d93eb	assistant	好的，下面这段内容大约500字，我尽量写得自然一些，方便你观察流式输出的效果。\n\n流式输出，简单来说，就是模型在生成内容时，不是等整段文字全部想好、一次性吐出来，而是一个字或一个词地逐步返回。你看到的文字像是从一条细管里慢慢流出来的水，而不是突然倒下一整桶。这种体验上的差别其实很微妙，但对用户来说却很重要。因为人在等待时，最怕的不是时间本身，而是“不知道还要等多久”。如果屏幕上一片空白，哪怕只过去两秒，也会觉得漫长；而如果文字已经开头，哪怕生成速度不快，你也会觉得系统正在工作，心理上的等待感就弱了很多。\n\n从技术角度看，流式输出背后通常涉及网络传输、缓冲策略和服务端推理的配合。服务端每生成一小段内容，就立刻通过连接推送给前端，前端再实时渲染到界面上。这里的关键在于“及时”和“稳定”之间的平衡。如果推送太碎，网络开销会增加；如果缓冲太多，又失去了流式的意义。所以工程上往往要设置合理的分块大小和刷新频率，让用户既感到连贯，又觉得响应迅速。\n\n对写作类应用来说，流式输出还有额外的好处。它让整个过程更像是一次“对话中的思考”，而不是一次冷冰冰的结果交付。你甚至可以边看边打断、边调整方向，这种交互感会更强。当然，流式输出也对前端提出了要求：要处理好光标位置、自动滚动、长文本性能，以及意外断开后的恢复问题。否则，流式反而可能带来卡顿或跳动，影响体验。\n\n总之，流式输出不只是一个技术细节，它直接改变了人与模型互动时的节奏和感受。好的流式体验，应该让人几乎忘记“等待”这件事，只觉得内容在自然生长。希望这段文字足够你观察逐字出现的节奏，也能顺便验证一下长文本下是否依然流畅。	2026-09-16 10:50:30.953388+00
10	38e3d66e-d629-4d9b-bd66-6da9ae02b5e9	user	hi	2026-09-16 12:50:05.982284+00
11	38e3d66e-d629-4d9b-bd66-6da9ae02b5e9	assistant	Hi! How can I help you today?	2026-09-16 12:50:06.674117+00
12	ed8674ad-99e9-463f-a9ce-720145148a0d	user	让我来测测你 你是干什么的？	2026-09-16 15:32:03.275744+00
13	ed8674ad-99e9-463f-a9ce-720145148a0d	assistant	你好，我是 **Kada 平台的 AI 助手**，主要帮你分析和处理**短链接数据**。\n\n我可以协助你：\n- 查看短链接的访问量、点击趋势\n- 分析流量来源、地域、设备等数据\n- 对比不同短链接表现\n- 提供链接管理和优化建议\n\n你可以直接发具体短链接或数据问题，我来帮你分析。	2026-09-16 15:32:03.286205+00
14	4673e2ae-b890-41a9-90a8-60aadcabeb05	user	用一句话介绍一下你自己	2026-09-16 15:56:56.51535+00
15	4673e2ae-b890-41a9-90a8-60aadcabeb05	assistant	我是kada平台的AI助手，专注帮你分析和管理短链接数据。	2026-09-16 15:56:56.525494+00
16	ed8674ad-99e9-463f-a9ce-720145148a0d	user	你好	2026-09-16 15:58:42.236042+00
17	ed8674ad-99e9-463f-a9ce-720145148a0d	assistant	你好！我是 Kada 平台的 AI 助手，可以帮你查看短链接的点击量、流量来源、地域、设备等数据。请告诉我你想分析哪个短链接，或有什么具体问题。	2026-09-16 15:58:42.242774+00
18	3817676e-8338-49c6-9102-ad17951fb56a	user	可以》	2026-09-17 06:26:05.779325+00
19	3817676e-8338-49c6-9102-ad17951fb56a	assistant	好的，请告诉我您需要分析或管理哪些短链接数据。	2026-09-17 06:26:05.789837+00
20	3817676e-8338-49c6-9102-ad17951fb56a	user	你知道你这个网页做什么的吗	2026-09-17 06:26:22.799887+00
21	3817676e-8338-49c6-9102-ad17951fb56a	assistant	知道。这个网页是 **Kada 平台**里的 AI 助手页面，主要用来帮助你 **分析和管理短链接数据**。\n\n我可以协助你：\n- 查看短链接的访问/点击数据\n- 分析点击趋势、来源等\n- 管理短链接相关信息\n\n如果你想了解具体功能，可以直接告诉我想做什么。	2026-09-17 06:26:22.806224+00
22	3817676e-8338-49c6-9102-ad17951fb56a	user	那查看短链接的访问	2026-09-17 06:26:45.483968+00
23	3817676e-8338-49c6-9102-ad17951fb56a	assistant	可以。请把要查看的 **短链接（或短码/名称）** 发给我，并说明时间范围（如今天、近7天、近30天、自定义）。\n\n我可以帮你查看/分析这些访问数据：\n- 总点击量、独立访客数\n- 按天/小时趋势\n- 访问来源（渠道、Referrer）\n- 地域、设备、浏览器分布\n- 最近访问记录\n\n如果是平台里已有的短链接，也可以在短链接列表中找到它，点击进入详情页查看访问数据。	2026-09-17 06:26:45.49084+00
24	06f6b042-ad13-4873-8bfc-23f32f6b57c3	user	今天是？	2026-09-17 10:50:39.635658+00
25	06f6b042-ad13-4873-8bfc-23f32f6b57c3	assistant	今天是2026年5月7日，星期四。	2026-09-17 10:50:39.648379+00
26	06f6b042-ad13-4873-8bfc-23f32f6b57c3	user	我上个问题是？	2026-09-17 10:50:49.689926+00
27	06f6b042-ad13-4873-8bfc-23f32f6b57c3	assistant	你上一个问题是：“今天是？”	2026-09-17 10:50:49.696296+00
28	06f6b042-ad13-4873-8bfc-23f32f6b57c3	user	今天怎么是5月7号	2026-09-17 10:51:29.700033+00
29	06f6b042-ad13-4873-8bfc-23f32f6b57c3	assistant	我并不能直接感知现实时间，刚才的“2026年5月7日”是根据当前对话环境提供的时间信息来回答的，不一定等于你所在时区的实际日期。\n\n如果你那边不是5月7日，请以你设备上的日历为准；你也可以告诉我正确日期，我按你给的日期来。	2026-09-17 10:51:29.7058+00
\.


--
-- Name: ai_messages_id_seq; Type: SEQUENCE SET; Schema: public; Owner: kada
--

SELECT pg_catalog.setval('public.ai_messages_id_seq', 29, true);


--
-- Name: ai_conversations ai_conversations_pkey; Type: CONSTRAINT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.ai_conversations
    ADD CONSTRAINT ai_conversations_pkey PRIMARY KEY (id);


--
-- Name: ai_messages ai_messages_pkey; Type: CONSTRAINT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.ai_messages
    ADD CONSTRAINT ai_messages_pkey PRIMARY KEY (id);


--
-- Name: ix_ai_conversations_user_id; Type: INDEX; Schema: public; Owner: kada
--

CREATE INDEX ix_ai_conversations_user_id ON public.ai_conversations USING btree (user_id);


--
-- Name: ix_ai_messages_conversation_id; Type: INDEX; Schema: public; Owner: kada
--

CREATE INDEX ix_ai_messages_conversation_id ON public.ai_messages USING btree (conversation_id);


--
-- Name: ai_messages ai_messages_conversation_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: kada
--

ALTER TABLE ONLY public.ai_messages
    ADD CONSTRAINT ai_messages_conversation_id_fkey FOREIGN KEY (conversation_id) REFERENCES public.ai_conversations(id) ON DELETE CASCADE;


--
-- Name: SCHEMA public; Type: ACL; Schema: -; Owner: pg_database_owner
--

GRANT CREATE ON SCHEMA public TO kada;


--
-- Name: DEFAULT PRIVILEGES FOR SEQUENCES; Type: DEFAULT ACL; Schema: public; Owner: postgres
--

ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA public GRANT ALL ON SEQUENCES TO kada;


--
-- Name: DEFAULT PRIVILEGES FOR TABLES; Type: DEFAULT ACL; Schema: public; Owner: postgres
--

ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA public GRANT ALL ON TABLES TO kada;


--
-- PostgreSQL database dump complete
--

\unrestrict eFzbyQgLqPHyNPS4G19vbLXThKUBdr2J15hykuIj7mqZImvV2mbZkBG9uXVsqrQ

--
-- Database "postgres" dump
--

--
-- PostgreSQL database dump
--

\restrict mJpqCzJSfnfyngzwDS3maWXkMVOrjsjnovJcxPFAtOj3eIAKAJxaurNKtefxPvh

-- Dumped from database version 16.15 (Debian 16.15-1.pgdg13+2)
-- Dumped by pg_dump version 16.15 (Debian 16.15-1.pgdg13+2)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

DROP DATABASE postgres;
--
-- Name: postgres; Type: DATABASE; Schema: -; Owner: postgres
--

CREATE DATABASE postgres WITH TEMPLATE = template0 ENCODING = 'UTF8' LOCALE_PROVIDER = libc LOCALE = 'en_US.utf8';


ALTER DATABASE postgres OWNER TO postgres;

\unrestrict mJpqCzJSfnfyngzwDS3maWXkMVOrjsjnovJcxPFAtOj3eIAKAJxaurNKtefxPvh
\connect postgres
\restrict mJpqCzJSfnfyngzwDS3maWXkMVOrjsjnovJcxPFAtOj3eIAKAJxaurNKtefxPvh

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: DATABASE postgres; Type: COMMENT; Schema: -; Owner: postgres
--

COMMENT ON DATABASE postgres IS 'default administrative connection database';


--
-- PostgreSQL database dump complete
--

\unrestrict mJpqCzJSfnfyngzwDS3maWXkMVOrjsjnovJcxPFAtOj3eIAKAJxaurNKtefxPvh

--
-- PostgreSQL database cluster dump complete
--

