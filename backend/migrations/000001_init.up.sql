
-- User table
create table
  if not exists dgv_user (
    id varchar(50) not null,
    address varchar(255) not null,
    email varchar(255),
    ctime timestamp default now (),
    utime timestamp,
    primary key (id)
  );

create unique index uq_dgv_user_address on dgv_user (address);

comment on table dgv_user is 'User table for degov apps';
comment on column dgv_user.address is 'wallet address';

-- DAO table
create table
  if not exists dgv_dao (
    id varchar(50) not null,
    chain_id int not null,
    chain_name varchar(255) not null,
    chain_logo text,
    name varchar(255) not null,
    code varchar(255) not null,
    logo text,
    seq int not null default 0,
    endpoint varchar(255) not null, -- website endpoint
    state varchar(50) not null, -- { ACTIVE, INACTIVE }
    tags text,
    config_link varchar(255) not null,
    time_syncd timestamp,
    metrics_count_proposals int not null default 0,
    metrics_count_members int not null default 0,
    metrics_sum_power varchar(255) not null default '0',
    metrics_count_vote int not null default 0,
    offset_tracking_proposal int default 0,
    ctime timestamp default now (),
    utime timestamp,
    primary key (id)
  );

create unique index uq_dgv_dao_code on dgv_dao (code);

comment on table dgv_dao is 'Available DAOs';
comment on column dgv_dao.chain_id is 'chain id';
comment on column dgv_dao.chain_name is 'chain name';
comment on column dgv_dao.name is 'DAO name';
comment on column dgv_dao.code is 'DAO code';
comment on column dgv_dao.tags is 'Optional tags for DAO categorization';
comment on column dgv_dao.time_syncd is 'last syncd time';

-- DAO chip table
create table
  if not exists dgv_dao_chip (
    id varchar(50) not null,
    dao_code varchar(255) not null,
    chip_code varchar(255) not null,
    flag varchar(255),
    additional text,
    ctime timestamp default now (),
    utime timestamp,
    primary key (id)
  );

create index idx_dgv_dao_chip_dao_code on dgv_dao_chip (dao_code);

-- DAO config table
create table
  if not exists dgv_dao_config (
    id varchar(50) not null,
    dao_code varchar(255) not null,
    config text not null,
    ctime timestamp default now (),
    utime timestamp,
    primary key (id)
  );

create unique index uq_dgv_dao_config_code on dgv_dao_config (dao_code);

comment on table dgv_dao_config is 'DAO config table';

-- User liked DAO table
create table
  if not exists dgv_user_liked_dao (
    id varchar(50) not null,
    dao_code varchar(255) not null,
    user_id varchar(50) not null,
    user_address varchar(255) not null,
    ctime timestamp default now (),
    primary key (id)
  );

create unique index uq_dgv_user_liked_dao_code_uid on dgv_user_liked_dao (dao_code, user_id);
create unique index uq_dgv_user_liked_dao_code_address on dgv_user_liked_dao (dao_code, user_address);

comment on table dgv_user_liked_dao is 'DAO like table';
comment on column dgv_user_liked_dao.dao_code is 'DAO code';
comment on column dgv_user_liked_dao.user_id is 'user id';
comment on column dgv_user_liked_dao.user_address is 'user address';

-- User subscribed DAO table (with notification updates)
create table
  if not exists dgv_user_subscribed_dao (
    id varchar(50) not null,
    chain_id int not null,
    dao_code varchar(255) not null,
    user_id varchar(50) not null,
    user_address varchar(255) not null,
    state varchar(50) not null, -- { ACTIVE, INACTIVE }
    ctime timestamp default now (),
    utime timestamp,
    primary key (id)
  );

create unique index uq_dgv_user_subscribe_uid_dao_code on dgv_user_subscribed_dao (user_id, dao_code);
create unique index uq_dgv_user_subscribe_address_dao_code on dgv_user_subscribed_dao (user_address, dao_code);

comment on table dgv_user_subscribed_dao is 'User subscribed DAO table';
comment on column dgv_user_subscribed_dao.user_id is 'user id';
comment on column dgv_user_subscribed_dao.user_address is 'user address';
comment on column dgv_user_subscribed_dao.dao_code is 'DAO code';

-- User subscribed proposal table (with notification updates)
create table
  if not exists dgv_user_subscribed_proposal (
    id varchar(50) not null,
    chain_id int not null,
    dao_code varchar(255) not null,
    user_id varchar(50) not null,
    user_address varchar(255) not null,
    state varchar(50) not null, -- { ACTIVE, INACTIVE }
    proposal_id varchar(255) not null,
    ctime timestamp default now (),
    utime timestamp,
    primary key (id)
  );

comment on table dgv_user_subscribed_proposal is 'User subscribed proposal table';
comment on column dgv_user_subscribed_proposal.user_address is 'user address';

-- Notification record table (with notification updates)
create table
  if not exists dgv_notification_record (
    id varchar(50) not null,
    code varchar(100) not null,
    chain_id int not null,
    dao_code varchar(255) not null,
    type varchar(50) not null, -- { PROPOSAL_NEW, PROPOSAL_STATE_CHANGED, VOTE_END, VOTE_EMITTED }
    user_id varchar(50) not null,
    user_address varchar(255) not null,
    message text,
    times_retry int not null default 0, -- number of times to retry sending
    proposal_id varchar(255) not null,
    vote_id varchar(255),
    state varchar(50) not null,
    payload text,
    event_id varchar(255) not null,
    time_next_execute timestamp not null,
    ctime timestamp default now (),
    primary key (id)
  );

create unique index uq_notification_record_code on dgv_notification_record(code);
create unique index uq_notification_record_event_id_user_id on dgv_notification_record (event_id, user_id);

comment on table dgv_notification_record is 'Notification record table';
comment on column dgv_notification_record.code is 'code';
comment on column dgv_notification_record.chain_id is 'chain id';
comment on column dgv_notification_record.dao_code is 'DAO code';
comment on column dgv_notification_record.type is 'notification type';
comment on column dgv_notification_record.user_id is 'user id';
comment on column dgv_notification_record.proposal_id is 'proposal id';
comment on column dgv_notification_record.vote_id is 'vote id';

-- User channel table
create table
  if not exists dgv_user_channel (
    id varchar(50) not null,
    user_id varchar(50) not null,
    user_address varchar(255) not null,
    verified int not null default 0, -- whether the channel is verified
    channel_type varchar(50) not null, -- { EMAIL, SMS, PUSH }
    channel_value varchar(500) not null, -- email address or phone number
    payload text, -- additional data for the channel
    ctime timestamp default now (),
    primary key (id)
  );

comment on table dgv_user_channel is 'Notification channel settings for users';
comment on column dgv_user_channel.user_id is 'user id';
comment on column dgv_user_channel.channel_type is 'notification channel type';
comment on column dgv_user_channel.channel_value is 'notification channel value (email or phone)';
comment on column dgv_user_channel.payload is 'additional data for the channel';

-- Proposal tracking table (with notification updates)
create table
  if not exists dgv_proposal_tracking (
    id varchar(50) not null,
    dao_code varchar(255) not null,
    chain_id int not null,
    proposal_link varchar(255) not null, -- link to the proposal
    proposal_id varchar(255) not null,
    state varchar(50) not null, -- { Pending, Active, Canceled, Defeated, Succeeded, Queued, Executed, Expired }
    proposal_at_block int not null, -- block number when the proposal was created
    proposal_created_at timestamp, -- proposal creation time
    times_track int not null default 0,
    time_next_track timestamp, -- next tracking time
    message text,
    offset_tracking_vote int default 0,
    title varchar(500) not null,
    ctime timestamp default now (),
    utime timestamp,
    primary key (id)
  );

comment on table dgv_proposal_tracking is 'DAO proposal tracking table';

-- Contracts ABI table (from migration 002)
create table
  if not exists dgv_contracts_abi (
    id varchar(50) not null,
    chain_id int not null,
    address varchar(255) not null,
    type varchar(30) not null,
    implementation varchar(255),
    abi text,
    ctime timestamp default now (),
    utime timestamp,
    primary key (id)
  );

-- Subscribed feature table (from migration 003)
create table
  if not exists dgv_subscribed_feature (
    id varchar(50) not null,
    chain_id int not null,
    dao_code varchar(255) not null,
    user_id varchar(50) not null,
    user_address varchar(255) not null,
    feature varchar(255) not null,
    strategy varchar(255) not null,
    proposal_id varchar(255),
    ctime timestamp default now (),
    primary key (id)
  );

comment on column dgv_subscribed_feature.feature is 'subscribe feature';
comment on column dgv_subscribed_feature.strategy is 'subscribe strategy';
comment on column dgv_subscribed_feature.proposal_id is 'proposal id';

-- Notification event table (from migration 003)
create table
  if not exists dgv_notification_event (
    id varchar(50) not null,
    chain_id int not null,
    dao_code varchar(255) not null,
    type varchar(50) not null,
    proposal_id varchar(255) not null,
    vote_id varchar(255),
    reached int not null default 0,
    state varchar(50) not null,
    times_retry int not null default 0,
    time_event timestamp,
    payload text,
    message text,
    ctime timestamp default now (),
    primary key (id)
  );

comment on column dgv_notification_event.type is 'event type';
