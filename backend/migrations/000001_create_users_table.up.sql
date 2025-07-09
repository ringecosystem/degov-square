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
