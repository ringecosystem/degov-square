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



-- create table if not exists dgv_dao(
--   id varchar(50) not null,

-- );

