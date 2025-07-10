create table
  if not exists dgv_user (
    id varchar(50) not null,
    address varchar(255) not null,
    email varchar(255),
    primary key (id)
  );

create unique index uq_dgv_user_address on dgv_user (address);

comment on table dgv_user is 'User table for degov apps';

comment on column dgv_user.address is 'wallet address';

create table
  if not exists dgv_dao (
    id varchar(50) not null,
    chain_id int not null,
    chain_name varchar(255) not null,
    name varchar(255) not null,
    code varchar(255) not null,
    config_link varchar(255) not null,
    time_sync timestamp,
    ctime timestamp default now (),
    primary key (id)
  );

create unique index uq_dgv_dao_code on dgv_dao (code);

comment on table dgv_dao is 'Available DAOs';

comment on column dgv_dao.chain_id is 'chain id';

comment on column dgv_dao.chain_name is 'chain name';

comment on column dgv_dao.name is 'DAO name';

comment on column dgv_dao.code is 'DAO code';

comment on column dgv_dao.config_link is 'DAO config link';

comment on column dgv_dao.time_sync is 'last sync time';

create table
  if not exists dgv_like_dao (
    id varchar(50) not null,
    dao_code varchar(50) not null,
    user_id varchar(50) not null,
    ctime timestamp default now (),
    primary key (id)
  );

create unique index uq_dgv_like_dao on dgv_like_dao (dao_code, user_id);

comment on table dgv_like_dao is 'DAO like table';

comment on column dgv_like_dao.dao_code is 'DAO code';

comment on column dgv_like_dao.user_id is 'user id';

create table
  if not exists dgv_user_followed_dao (
    id varchar(50) not null,
    chain_id int not null,
    dao_code varchar(50) not null,
    user_id varchar(50) not null,
    enable_new_proposal int not null default 1,
    enable_voting_end_reminder int not null default 0,
    ctime timestamp default now (),
    primary key (id)
  );

create unique index uq_dgv_notification on dgv_notification (user_id, dao_code);

comment on table dgv_notification is 'Notification settings for users';

comment on column dgv_notification.user_id is 'user id';

comment on column dgv_notification.dao_code is 'DAO code';

comment on column dgv_notification.enable_new_proposal is 'enable new proposal notification';

comment on column dgv_notification.enable_voting_end_reminder is 'enable voting end reminder';

create table
  if not exists dgv_notification_record (
    id varchar(50) not null,
    chain_id int not null,
    chain_name varchar(255) not null,
    dao_name varchar(255) not null,
    dao_code varchar(50) not null,
    type varchar(50) not null, -- { NEW_PROPOSAL, VOTE, STATUS, VOTE_END_REMINDER }
    target_id varchar(255), -- proposal id or vote id
    user_id varchar(50) not null,
    status varchar(50) not null, -- { SENT_OK, SENT_FAIL }
    message text,
    retry_times int not null default 0, -- number of times to retry sending
    ctime timestamp default now (),
    primary key (id)
  );

comment on table dgv_notification_record is 'Notification record table';

comment on column dgv_notification_record.chain_id is 'chain id';

comment on column dgv_notification_record.chain_name is 'chain name';

comment on column dgv_notification_record.dao_name is 'DAO name';

comment on column dgv_notification_record.dao_code is 'DAO code';

comment on column dgv_notification_record.type is 'notification type';

comment on column dgv_notification_record.target_id is 'target id (proposal id or vote id)';

comment on column dgv_notification_record.user_id is 'user id';

comment on column dgv_notification_record.status is 'notification status';

create table
  if not exists dgv_user_channel (
    id varchar(50) not null,
    user_id varchar(50) not null,
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
