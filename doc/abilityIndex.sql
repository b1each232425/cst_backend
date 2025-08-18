/*==============================================================*/
/* DBMS name:      PostgreSQL 9.x                               */
/* Created on:     2025/8/18 17:00:13                           */
/*==============================================================*/


drop view if exists v_z_practice_wrong_collection;

drop view if exists v_z_practice_summary;

drop view if exists v_z_grade_practice_statistics;

drop view if exists v_z_grade_exam_session_info;

drop view if exists v_y_max_submitted_view;

drop view if exists v_report_claims;

drop view if exists v_mistake_correct;

drop view if exists v_mistake_correct2;

drop view if exists v_xkb_user;

drop view if exists v_xkb_school_layout;

drop view if exists v_x_grade_list;

drop view if exists v_user_domain_api;

drop view if exists v_user_domain;

drop view if exists v_user;

drop view if exists v_student_practice_total_score;

drop view if exists v_student_exam_total_score;

drop view if exists v_student_answer_question;

drop view if exists v_region;

drop view if exists v_question_bank;

drop view if exists v_practice_unmarked_student_cnt;

drop view if exists v_payment;

drop view if exists v_param;

drop view if exists v_paper;

drop view if exists v_order_sum;

drop view if exists v_order2;

drop view if exists v_order;

drop view if exists v_mistake_correct_show;

drop view if exists v_manager_school;

drop view if exists v_latest_unsubmitted_practice;

drop view if exists v_latest_submitted_practice;

drop view if exists v_latest_pending_mark_practice;

drop view if exists v_invigilation_info;

drop view if exists v_insurer;

drop view if exists v_insured_school;

drop view if exists v_insure_attach;

drop view if exists v_insurance_type;

drop view if exists v_insurance_policy2;

drop view if exists v_insurance_policy;

drop view if exists v_examinee_info;

drop view if exists v_exam_unmarked_student_count;

drop view if exists v_exam_teacher_marked_count;

drop view if exists v_exam_respondent_count;

drop view if exists v_exam_paper;

drop view if exists v_exam_file;

drop view if exists v_domain_user;

drop view if exists v_domain_asset;

drop view if exists v_domain_api;

drop view if exists v_authenticate;

drop view if exists v_api_domain;

drop view if exists v_aa;

drop table if exists t_account_opr_log;

drop table if exists t_age;

drop index if exists  idx_t_api_name;

drop index if exists  idx_t_api_expose_path;

drop table if exists t_domain_api;

drop index if exists  idx_api_domain;

drop table if exists t_api;

drop table if exists t_article;

drop table if exists t_blacklist;

drop table if exists t_course;

drop table if exists t_degree;

drop index if exists  idx_domain_domain;

drop table if exists t_user_domain;

drop index if exists  idx_user_domain;

drop table if exists t_domain;

drop index if exists  app_user_id_idx;

drop index if exists  domain_asset_relation;

drop table if exists t_domain_asset;

drop table if exists t_exam_info;

drop table if exists t_exam_paper;

drop table if exists t_exam_paper_group;

drop table if exists t_exam_paper_question;

drop table if exists t_exam_record;

drop table if exists t_exam_room;

drop table if exists t_exam_session;

drop table if exists t_exam_site;

drop table if exists t_examinee;

drop table if exists t_expertise;

drop table if exists t_external_domain_conf;

drop index if exists  idx_ext_user_domain_b_d_u;

drop table if exists t_external_domain_user;

drop table if exists t_file;

drop table if exists t_group;

drop index if exists  idx_impdata_entity_id;

drop index if exists  idx_impdata_digest;

drop index if exists  idx_impdata_key;

drop table if exists t_import_data;

drop index if exists  idx_insure_policy_status;

drop index if exists  idx_insure_policy_order_id;

drop index if exists  idx_insure_policy_SN;

drop table if exists t_insurance_policy;

drop index if exists  idx_insure_type_channel;

drop index if exists  idx_insure_type_refid_orgid;

drop table if exists t_insurance_types;

drop index if exists  idx_insure_attach_p;

drop index if exists  idx_insure_attach;

drop table if exists t_insure_attach;

drop index if exists  idx_insured_detail_o_p_n;

drop table if exists t_insured_detail;

drop table if exists t_insured_terms;

drop table if exists t_invigilation;

drop table if exists t_judge;

drop index if exists  idx_t_log_base;

drop index if exists  t_log_PK;

drop table if exists t_log;

drop table if exists t_mark;

drop table if exists t_mark_info;

drop table if exists t_mistake_correct;

drop table if exists t_msg;

drop table if exists t_msg_status;

drop table if exists t_my_contact;

drop index if exists  idx_negotiation;

drop table if exists t_negotiated_price;

drop index if exists  idx_t_order_agency_id;

drop index if exists  idx_t_order_create_time;

drop index if exists  idx_t_order_trade_no2;

drop table if exists t_order;

drop table if exists t_paper;

drop table if exists t_paper_group;

drop table if exists t_paper_question;

drop index if exists  t_param_full_idx;

drop table if exists t_param;

drop index if exists  account_name;

drop index if exists  account_info;

drop table if exists t_pay_account;

drop table if exists t_payment;

drop table if exists t_practice;

drop table if exists t_practice_student;

drop table if exists t_practice_submissions;

drop table if exists t_price;

drop table if exists t_prj;

drop table if exists t_proof;

drop table if exists t_prove;

drop table if exists t_qualification;

drop index if exists  idx_question_id_creator;

drop index if exists  idx_question_id;

drop table if exists t_question;

drop index if exists  idx_question_bank_id_creator;

drop index if exists  idx_question_bank_id;

drop table if exists t_question_bank;

drop index if exists  idx_region_name;

drop index if exists  Idx_region_id_pid;

drop table if exists t_region;

drop index if exists  idx_key_key_relation;

drop index if exists  idx_id_key_relation;

drop index if exists  idx_key_id_relation;

drop index if exists  idx_id_id_relation;

drop table if exists t_relation;

drop index if exists  idx_relation_history;

drop table if exists t_relation_history;

drop table if exists t_report_claims;

drop table if exists t_resource;

drop table if exists t_scan_tdc;

drop index if exists  idx_school_name;

drop table if exists t_school;

drop table if exists t_section;

drop index if exists  special_order_open_id;

drop index if exists  special_order_prj_id;

drop table if exists t_special_order;

drop table if exists t_student_answers;

drop table if exists t_sys_ver;

drop table if exists t_tdc;

drop table if exists t_teacher_student;

drop table if exists t_undertaker;

drop index if exists  idx_user_offcial_name;

drop index if exists  idx_user_id_card_no;

drop index if exists  idx_user_email;

drop index if exists  idx_user_phone;

drop index if exists  idx_user_nickName;

drop index if exists  idx_user_externalID;

drop index if exists  idx_user_account;

drop table if exists t_wx_user;

drop index if exists  idx_wx_user_full;

drop index if exists  idx_wx_user_openid;

drop table if exists t_xkb_user;

drop table if exists t_user;

drop table if exists t_user_assessment;

drop index if exists  idx_user_course_u_c;

drop table if exists t_user_course;

drop table if exists t_user_degree;

drop index if exists  idx_user_grp;

drop table if exists t_user_group;

/*==============================================================*/
/* Table: t_account_opr_log                                     */
/*==============================================================*/
create table if not exists  t_account_opr_log (
   id                   SERIAL not null,
   user_id              INT8                 null,
   original             JSONB                null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   creator              INT8                 null,
   domain_id            INT8                 null,
   addi                 JSONB                null,
   remark               VARCHAR              null,
   constraint PK_T_ACCOUNT_OPR_LOG primary key (id)
);

comment on table t_account_opr_log is
'用户关键信息变更日志，谁，在什么时间变更了数据，变更前数据是什么样子';

comment on column t_account_opr_log.id is
'操作编号';

comment on column t_account_opr_log.user_id is
'被变更用户编号';

comment on column t_account_opr_log.original is
'原账号数据';

comment on column t_account_opr_log.create_time is
'生成时间';

comment on column t_account_opr_log.creator is
'本数据创建者';

comment on column t_account_opr_log.domain_id is
'数据隶属';

comment on column t_account_opr_log.addi is
'附加信息';

comment on column t_account_opr_log.remark is
'备注';

/*==============================================================*/
/* Table: t_age                                                 */
/*==============================================================*/
create table if not exists  t_age (
   id                   SERIAL not null,
   insurance_type_id    INT8                 null,
   school_type          VARCHAR              null,
   province             VARCHAR              null,
   city                 VARCHAR              null,
   district             VARCHAR              null,
   enabled              BOOL                 null default true,
   male_max             INT2                 null,
   male_min             INT2                 null,
   female_min           INT2                 null,
   female_max           INT2                 null,
   domain_id            INT8                 null,
   addi                 JSONB                null,
   creator              INT8                 null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   updated_by           INT8                 null,
   remark               VARCHAR              null,
   status               VARCHAR              null default '0',
   constraint PK_T_AGE primary key (id)
);

comment on table t_age is
'年龄表';

comment on column t_age.id is
'条目编号';

comment on column t_age.insurance_type_id is
'保险类型';

comment on column t_age.school_type is
'学校类别，可以多个类别，用空格间隔开识别';

comment on column t_age.province is
'省';

comment on column t_age.city is
'市';

comment on column t_age.district is
'区/县';

comment on column t_age.enabled is
'是否开启限制年龄；true开启限制年龄，false关闭限制年龄';

comment on column t_age.male_max is
'男性年龄最大值';

comment on column t_age.male_min is
'男性年龄最小值';

comment on column t_age.female_min is
'女性年龄最小值';

comment on column t_age.female_max is
'女性年龄最大值';

comment on column t_age.domain_id is
'数据属主';

comment on column t_age.addi is
'备用字段';

comment on column t_age.creator is
'创建者用户ID';

comment on column t_age.create_time is
'创建时间';

comment on column t_age.update_time is
'更新时间';

comment on column t_age.updated_by is
'更新人';

comment on column t_age.remark is
'备注';

comment on column t_age.status is
'0:有效, 2: 删除';

ALTER SEQUENCE t_age_id_seq RESTART WITH 20000;
insert into t_age
  (id, insurance_type_id, enabled, male_max, male_min, female_max,female_min)
  values(1000, 10000, true,65, 2, 55,2);
insert into t_age
  (id, insurance_type_id, enabled, male_max, male_min, female_max,female_min)
  values(1002, 10022, true,60, 2, 55,2);
  insert into t_age
  (id, insurance_type_id, enabled, male_max, male_min, female_max,female_min)
  values(1004, 10024, true,60,2, 55,2);
    insert into t_age
  (id, insurance_type_id, enabled, male_max, male_min, female_max,female_min)
  values(1006, 10026, true,60,2, 55,2);
  
  insert into t_age
  (id, insurance_type_id, enabled, male_max, male_min, female_max,female_min)
  values(1010, 10040, true,65, 2, 55,2);

/*==============================================================*/
/* Table: t_api                                                 */
/*==============================================================*/
create table if not exists  t_api (
   id                   SERIAL not null,
   name                 VARCHAR              not null,
   expose_path          VARCHAR              null,
   maintainer           INT8                 null,
   access_control_level VARCHAR              not null,
   updated_by           INT8                 null,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   creator              INT8                 null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   domain_id            INT8                 null,
   addi                 JSONB                null,
   Remark               VARCHAR              null,
   status               VARCHAR              null default '01',
   constraint PK_T_API primary key (id)
);

comment on table t_api is
'接口信息表';

comment on column t_api.id is
'编码';

comment on column t_api.name is
'接口名称';

comment on column t_api.expose_path is
'访问路径';

comment on column t_api.maintainer is
'维护者';

comment on column t_api.access_control_level is
'访问控制实现层级
level 0: 无组/角色/数据限制
level 2: 机构#角色级别, 实现了不同角色授权，但不控制数据范围
level 4: 机构#角色$ID, 实现了不同角色授权，可控制 creator || all
level 8: 机构.DEPT#角色$ID, 实现了不同角色授权，可控制 creator || GRPs';

comment on column t_api.updated_by is
'更新者';

comment on column t_api.update_time is
'帐号信息更新时间';

comment on column t_api.creator is
'本数据创建者';

comment on column t_api.create_time is
'生成时间';

comment on column t_api.domain_id is
'数据隶属';

comment on column t_api.addi is
'附加信息';

comment on column t_api.Remark is
'备注';

comment on column t_api.status is
'状态，00：草稿，01：有效，02：作废';

ALTER SEQUENCE t_api_id_seq RESTART WITH 20000;
/*
insert into t_api(id,name, expose_path,domain_id, maintainer, access_control_level)
values (100,'平台.登录','/api/login',177,1000,3),
(110,'网站图标','/favicon.ico',177,1000,3),
(200,'平台.登出','/api/logout',177,1000,3),
(300,'平台.用户管理','/api/user',177,1000,3),
(400,'校快保.参数','/api/param',10002,1000,3),
(500,'校快保.学校','/api/school',10002,1000,3),
(600,'校快保.开放学校列表','/api/openSchools',10002,1000,3),
(700,'校快保.学校列表','/api/schoolList',10002,1000,3),
(800,'校快保.用户','/api/xkbUser',10002,1000,3),
(900,'校快保.我的被保险人','/api/myInsuredList',10002,1000,3),
(1000,'校快保.订单','/api/order',10002,1000,3),
(1100,'校快保.微信支付','/api/wxPay',10002,1000,3),
(1200,'校快保.微信支付回调','/api/wxPaid',10002,1000,3),
(1300,'平台.时间测试','/api/trialTime',177,1000,3),
(1400,'校快保.投保规则','/api/purchaseRule',10002,1000,3),
(1500,'校快保.微信支付参数','/api/wxAppID',10002,1000,3),
(1600,'校快保.用户状态','/api/status',10002,1000,3),
(1700,'平台.微信消息响应','/api/wxServe',177,1000,3),
(1800,'平台.微信登录','/api/wxLogin',177,1000,3),
(1900,'近邻科技.微信验证域名','/MP_verify_NoNLb44EuoLJ7ybT.txt',1077,1000,3),
(2000,'校快保.微信验证域名','/MP_verify_87DVhsMdnS64dC0K.txt',10002,1000,3),
(2100,'平台.基础测试','/trial',177,1000,3),
(2200,'校快保.学校管理员','/api/schoolManager',10002,1000,3),
(2300,'校快保.销售管理员','/api/saleManager',10002,1000,3),
(2400,'校快保.学校相关管理员','/api/manager',10002,1000,3),
(2500,'校快保.保单','/api/insurancePolicy',10002,1000,3),
(2600,'校快保.保险单价','/api/insuranceUnitPrice',10002,1000,3),
(2700,'校快保.保险文档','/api/insuranceDoc',10002,1000,3),
(2800,'平台.发送短信验证码','/api/sendVerifyCodeBySMS',177,1000,3),
(2900,'平台.确认短信验证码','/api/verifySMSCode',177,1000,3),
(3000,'平台.手机是否已被验证','/api/isTelVerified',177,1000,3),
(3100,'平台.日志','/api/log',177,1000,3),
(3200,'平台.文件','/api/file',177,1000,3),
(3300,'校快保.报案理赔','/api/reportClaims',10002,1000,3),
(3400,'平台.默认页面','/',177,1000,3),
(3500,'平台.接口测试','/t',177,1000,3),
(3600,'平台.测试.登录页面','/t/login',177,1000,3),
(3700,'校快保.前端','/xkb',10002,1000,3),
(3800,'校快保.前端.登录','/xkb/login',10002,1000,3);
*/

/*==============================================================*/
/* Index: idx_t_api_expose_path                                 */
/*==============================================================*/
create unique index if not exists  idx_t_api_expose_path on t_api (
expose_path
);

/*==============================================================*/
/* Index: idx_t_api_name                                        */
/*==============================================================*/
create unique index if not exists  idx_t_api_name on t_api (
name
);

/*==============================================================*/
/* Table: t_article                                             */
/*==============================================================*/
create table if not exists  t_article (
   id                   SERIAL not null,
   author               VARCHAR              null,
   title                VARCHAR              null,
   subtitle             VARCHAR              null,
   keyword              VARCHAR              null,
   belong               INT8                 null,
   channel              JSONB                null,
   type                 JSONB                null,
   domain               JSONB                null,
   quality              INT4                 null,
   viewed               INT4                 null,
   score                JSONB                null,
   prosecute            JSONB                null,
   assent_num           INT4                 null,
   oppose_num           INT4                 null,
   source               VARCHAR              null,
   tags                 VARCHAR              null,
   face_pic_num         INT2                 null,
   content              JSONB                null,
   files                JSONB                null,
   creator              INT8                 null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   updated_by           INT8                 null,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   domain_id            INT8                 null,
   addi                 JSONB                null,
   remark               VARCHAR              null,
   status               VARCHAR              null,
   constraint PK_T_ARTICLE primary key (id)
);

comment on table t_article is
'消息：包含新闻，私信，广告，通知等';

comment on column t_article.id is
'参数编号';

comment on column t_article.author is
'作者';

comment on column t_article.title is
'标题';

comment on column t_article.subtitle is
'副标题';

comment on column t_article.keyword is
'关键字';

comment on column t_article.belong is
'属于';

comment on column t_article.channel is
'频道';

comment on column t_article.type is
'内容类型：搞笑，新闻，';

comment on column t_article.domain is
'领域：教育/游戏/电子等';

comment on column t_article.quality is
'内容质量';

comment on column t_article.viewed is
'阅读次数';

comment on column t_article.score is
'读者评分';

comment on column t_article.prosecute is
'举报';

comment on column t_article.assent_num is
'赞同数';

comment on column t_article.oppose_num is
'反对数';

comment on column t_article.source is
'来源';

comment on column t_article.tags is
'标签';

comment on column t_article.face_pic_num is
'封面图片数';

comment on column t_article.content is
'内容';

comment on column t_article.files is
'附加文件';

comment on column t_article.creator is
'本数据创建者';

comment on column t_article.create_time is
'生成时间';

comment on column t_article.updated_by is
'更新者';

comment on column t_article.update_time is
'修改时间';

comment on column t_article.domain_id is
'数据隶属';

comment on column t_article.addi is
'附加信息';

comment on column t_article.remark is
'备注';

comment on column t_article.status is
'状态，00：草稿，01：有效，02：作废';

ALTER SEQUENCE t_article_id_seq RESTART WITH 20000;

/*==============================================================*/
/* Table: t_blacklist                                           */
/*==============================================================*/
create table if not exists  t_blacklist (
   id                   SERIAL not null,
   order_id             INT8                 null,
   type                 VARCHAR              not null,
   content              VARCHAR              not null,
   updated_by           INT8                 null,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   creator              INT8                 null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   domain_id            INT8                 null,
   addi                 JSONB                null,
   status               VARCHAR              null,
   constraint PK_T_BLACKLIST primary key (id)
);

comment on table t_blacklist is
'黑名单表';

comment on column t_blacklist.id is
'拒保黑名单编号';

comment on column t_blacklist.order_id is
'来源订单号';

comment on column t_blacklist.type is
'黑名单类型（投保人，统一社会信用代码（投保人），投保联系人手机号码）';

comment on column t_blacklist.content is
'黑名单内容';

comment on column t_blacklist.updated_by is
'更新者';

comment on column t_blacklist.update_time is
'更新时间';

comment on column t_blacklist.creator is
'创建者用户ID';

comment on column t_blacklist.create_time is
'创建时间';

comment on column t_blacklist.domain_id is
'数据属主';

comment on column t_blacklist.addi is
'附加数据';

comment on column t_blacklist.status is
'状态 0:有效, 2: 无效';

ALTER SEQUENCE t_blacklist_id_seq RESTART WITH 20000;

/*==============================================================*/
/* Table: t_course                                              */
/*==============================================================*/
create table if not exists  t_course (
   id                   SERIAL not null,
   type                 VARCHAR              null,
   category             VARCHAR              null,
   name                 VARCHAR              null,
   issuer               VARCHAR              null,
   issue_time           INT8                 null,
   cover                JSONB                null,
   repos                JSONB                null,
   sections             JSONB                null,
   tags                 JSONB                null,
   data                 JSONB                null,
   default_repo         VARCHAR              null,
   creator              INT8                 not null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   updated_by           INT8                 null,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   domain_id            INT8                 null,
   addi                 JSONB                null,
   remark               VARCHAR              null,
   status               VARCHAR              null,
   constraint PK_T_COURSE primary key (id)
);

comment on table t_course is
'course table';

comment on column t_course.id is
'编码';

comment on column t_course.type is
'类型';

comment on column t_course.category is
'分类';

comment on column t_course.name is
'名称';

comment on column t_course.issuer is
'发布者';

comment on column t_course.issue_time is
'发布时间';

comment on column t_course.cover is
'封面介绍';

comment on column t_course.repos is
'仓库';

comment on column t_course.sections is
'章节列表';

comment on column t_course.tags is
'标签';

comment on column t_course.data is
'附加数据';

comment on column t_course.default_repo is
'课程git repo';

comment on column t_course.creator is
'创建者';

comment on column t_course.create_time is
'创建时间';

comment on column t_course.updated_by is
'更新者';

comment on column t_course.update_time is
'更新时间';

comment on column t_course.domain_id is
'数据隶属';

comment on column t_course.addi is
'用户定制数据';

comment on column t_course.remark is
'备注';

comment on column t_course.status is
'00：草稿
02：发布/上架
04：下架
06：禁用';

ALTER SEQUENCE t_course_id_seq RESTART WITH 20000;

/*==============================================================*/
/* Table: t_degree                                              */
/*==============================================================*/
create table if not exists  t_degree (
   id                   SERIAL not null,
   level                integer              null,
   name                 VARCHAR              null,
   limn                 VARCHAR              null,
   status               VARCHAR              null,
   constraint PK_T_DEGREE primary key (id)
);

comment on table t_degree is
'知识能力领域等级表';

comment on column t_degree.id is
'编号';

comment on column t_degree.level is
'等级';

comment on column t_degree.name is
'等级名称';

comment on column t_degree.limn is
'等级描述';

comment on column t_degree.status is
'可用，禁用';

ALTER SEQUENCE t_degree_id_seq RESTART WITH 10000;

/*==============================================================*/
/* Table: t_domain                                              */
/*==============================================================*/
create table if not exists  t_domain (
   id                   SERIAL not null,
   name                 VARCHAR              not null,
   domain               VARCHAR              not null,
   priority             INT2                 null,
   domain_id            INT8                 not null default 0,
   updated_by           INT8                 null,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   creator              INT8                 null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   addi                 JSONB                null,
   remark               VARCHAR              null,
   status               VARCHAR              null default '01',
   constraint PK_T_DOMAIN primary key (id)
);

comment on table t_domain is
'用户组织结构定义，格式为：机构[部门.科室.组]#角色';

comment on column t_domain.id is
'编码';

comment on column t_domain.name is
'域名称';

comment on column t_domain.domain is
'机构[部门.科室.组]^角色!userID';

comment on column t_domain.priority is
'0: 超级管理员, 可做任何事; 3: 普通管理员, 可做所属子系统的任何事; 5: 业务员, 可做一般的管理任务; 7: 普通用户, 只能访问自己的数据; 9: 匿名用户, 只能访问白名单功能。';

comment on column t_domain.domain_id is
'数据隶属';

comment on column t_domain.updated_by is
'更新者';

comment on column t_domain.update_time is
'帐号信息更新时间';

comment on column t_domain.creator is
'本数据创建者';

comment on column t_domain.create_time is
'生成时间';

comment on column t_domain.addi is
'附加信息';

comment on column t_domain.remark is
'备注';

comment on column t_domain.status is
'状态，00：草稿，01：有效，02：作废';

ALTER SEQUENCE t_domain_id_seq RESTART WITH 20000;

-- ***无角色的域仅做为数据隔离条件使用, 优先级为0, 不允许使用于账号授权**
-- sys是平台域
insert into t_domain(id,name, domain, creator,priority) values

(322,'系统','sys',1000,0),-- 表示该对对象是平台相关，非特定业务相关, 
(333,'系统.管理','sys^admin',1000,11),-- 平台管理员角色，上帝角色
(366,'系统.运维','sys^maintain',1000,13), -- 平台维护角色

-- (377,'系统.用户','sys^user',1000,17), -- 普通平台账号，这个不应该存在或被使用

(388,'系统.匿名','sys^anonymous',1000,19), -- 普通平台匿名账号，用户处于未登录状态时所属于的领域/角色

-- (561,'系统.运营','sys^promotion',1000,21), -- 普通平台运营角色账号, 这个不应该存在或被使用
-- (563,'系统.销售','sys^sale',1000,23), -- 普通平台销售角色账号，这个不应该存在或被使用

-- sys^trial 平台测试账号，相当于sys@admin，这个账号应该仅存在于开发数据库中
--     或以调试为目的短暂存在于生产系统中，该角色应该仅用于平台自身，用完即禁。每个应用应该建立自己的[app]^trial域
(566,'系统.测试','sys^trial',1000,9), 

-- common是为第三方提供服务的域,只能用于功能的domain属性, 表示该功
--     能可以跨域访问数据，数据约束则以账号的domain为线索。
(567,'管理','common^admin',1000,25),
(569,'运维','common^maintain',1000,27),


(671,'用户','common^user',1000,29),
(673,'匿名','common^anonymous',1000,31),

(675,'运营','common^promotion',1000,33),
(677,'销售','common^sale',1000,35),
(679,'测试','common^trial',1000,37),

-- 
(1077,'近邻科技','qnear',1000,0),
(1079,'近邻科技.管理','qnear^admin',1000,1005),

--
(1177,'能力索引','abilityIdx',1000,0),
(1179,'能力索引.管理','abilityIdx^admin',1000,1009),

--
(1277,'IT双创精英孵化实训室','foreseeLab',1000,0),
(1279,'IT双创精英孵化实训室.管理','foreseeLab^admin',1000,1019),

--
(1377,'人才引进','recuitMgr',1000,0),
(1379,'人才引进.管理','recuitMgr^admin',1000,1025),

--
(1477,'教学督导','jxdd',1000,0),
(1479,'教学督导.管理','jxdd^admin',1000,1035),

--
(1577,'校友会小额捐献','donate',1000,0),
(1579,'校友会小额捐献.管理','donate^admin',1000,1045),

--
(10002,'校快保','xkb',1000,0),
(10004,'校快保.管理','xkb^admin',1000,1055),
(10006,'校快保.销售经理','xkb^sale',1000,1057),
(10008,'校快保.学校管理员','xkb.school^admin',1000,1059),
(10010,'校快保.学校统计员','xkb.school^statistics',1000,1061),
(10012,'校快保.客户','xkb^user',1000,1067),
(10016,'校快保.运营','xkb^promotion',1000,1069),
(10020,'校快保.前台','xkb^fe',1000,1071),
(10030,'校快保.后台','xkb^be',1000,1073),

(10098,'考试系统','assess',1000,0),
(10100,'考试系统.学校管理员','assess^admin',1000,1083),
(10102,'考试系统.学校领导','assess^leader',1000,1085),
(10104,'考试系统.学院管理员','assess.faculty^admin',1000,1087),
(10106,'考试系统.教务处领导','assess.academicAffair^dean',1000,1089),
(10108,'考试系统.学生处领导','assess.studentAffair^dean',1000,1091),
(10110,'考试系统.学院领导','assess.faculty^leader',1000,1093),

(10112,'考试系统.教务员','assess.academicAffair^admin',1000,1095),
(10114,'考试系统.教师','assess^teacher',1000,1097),
(10116,'考试系统.监考员','assess^examSupervisor',1000,1099),
(10118,'考试系统.批阅员','assess^examGrader', 1000, 1111),
(10120,'考试系统.核分员','assess^scoreChecker', 1000, 1113),
(10122,'考试系统.考点','assess.examSite', 1000, 1115),
(10124,'考试系统.考点负责人','assess.examSite^Admin', 1000, 1117),
(10126,'考试系统.运维','assess^maintain',1000,1119), -- 平台维护角色
(10128,'考试系统.学生','assess^student',1000,1121),


(10200,'教学系统','course',1000,0),
(10202,'教学系统.管理员','course^admin',1000,1201),
(10204,'教学系统.运维','course^maintain',1000,1203), -- 平台维护角色
(10206,'教学系统.运营','course^promotion',1000,1205),
(10208,'教学系统.教师','course^teacher',1000,1207),
(10210,'教学系统.助教','course^teachingAssistant',1000,1209),
(10212,'教学系统.班长','course^classRepresentative',1000,1211),
(10214,'教学系统.学生','course^student',1000,1213);

/*==============================================================*/
/* Index: idx_domain_domain                                     */
/*==============================================================*/
create unique index if not exists  idx_domain_domain on t_domain (
domain
);

/*==============================================================*/
/* Table: t_domain_api                                          */
/*==============================================================*/
create table if not exists  t_domain_api (
   id                   SERIAL not null,
   api                  INT8                 not null,
   domain               INT8                 not null,
   grant_source         VARCHAR              null,
   data_access_mode     VARCHAR              null,
   data_scope           JSONB                null,
   domain_id            INT8                 null,
   updated_by           INT8                 null,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   creator              INT8                 null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   addi                 JSONB                null,
   remark               VARCHAR              null,
   status               VARCHAR              null default '01',
   constraint PK_T_DOMAIN_API primary key (id)
);

comment on table t_domain_api is
'用户、接口、数据访问控制表';

comment on column t_domain_api.id is
'权限编码';

comment on column t_domain_api.api is
'接口/功能编码';

comment on column t_domain_api.domain is
'组、角色编码';

comment on column t_domain_api.grant_source is
'grant:数据权限由t_relation中left_type:t_domain.id与left_type:t_user.id获得的数据决定,或data_scope中数据决定，但data_scope与t_relation只能存在一种，如果data_scope有效，则忽略t_relation;

cousin:忽略data_scope与t_relation, 授权数据由被过虑数据的domain_id决定,即被过虑数据的domain_id 与登录用户的t_user.domain_id相同或级别更低的数据，例如
    用户的t_user.domain为xkb^admin而数据的domain为xkb.school^admin，则用户可以获得该数据

self: 被过虑数据的creator 与登录用户的t_user.id相同

api: 由功能(api)自己决定 ';

comment on column t_domain_api.data_access_mode is
'数据访问类型, full:可读写, read: 只读, write: 写, partial: 部分写/混合';

comment on column t_domain_api.data_scope is
'当grant_source是grant时,以json数据方式提供数据授权范围格式为:
  {"granter":"t_user.id","grantee":"t_school.id","data":[1234,456,789]}
granter: 代表数据拥有者, t_user.id代表用户, t_domain.id代表角色,t_api.id代表功能
grantee: 代表拥有的数据,t_school.id代表可以访问的机构列表。
授权数据如果存储在t_relation中则各项分别对应如下
granter对应left_type, left_key对应t_user_domain.sys_user或t_domain_api.domain
grantee对应right_type, right_key对应right_type的意义';

comment on column t_domain_api.domain_id is
'数据领域归属';

comment on column t_domain_api.updated_by is
'更新者';

comment on column t_domain_api.update_time is
'帐号信息更新时间';

comment on column t_domain_api.creator is
'本数据创建者';

comment on column t_domain_api.create_time is
'生成时间';

comment on column t_domain_api.addi is
'附加信息';

comment on column t_domain_api.remark is
'备注';

comment on column t_domain_api.status is
'状态，00：草稿，01：有效，02：作废';

ALTER SEQUENCE t_domain_api_id_seq RESTART WITH 20000;

/*==============================================================*/
/* Index: idx_api_domain                                        */
/*==============================================================*/
create unique index if not exists  idx_api_domain on t_domain_api (
api,
domain
);

/*==============================================================*/
/* Table: t_user                                                */
/*==============================================================*/
create table if not exists  t_user (
   id                   SERIAL not null,
   external_id_type     VARCHAR              null,
   external_id          VARCHAR              null,
   category             VARCHAR              not null,
   type                 VARCHAR              null,
   language             VARCHAR              null,
   country              VARCHAR              null,
   province             VARCHAR              null,
   city                 VARCHAR              null,
   addr                 VARCHAR              null,
   fuse_name            VARCHAR              null,
   official_name        VARCHAR              null,
   id_card_type         VARCHAR              null,
   id_card_no           VARCHAR              null,
   mobile_phone         VARCHAR              null,
   email                VARCHAR              null,
   account              VARCHAR              not null,
   gender               VARCHAR              null,
   birthday             INT8                 null,
   nickname             VARCHAR              null,
   avatar               bytea                null,
   avatar_type          VARCHAR              null,
   dev_id               VARCHAR              null,
   dev_User_id          VARCHAR              null,
   dev_account          VARCHAR              null,
   cert                 VARCHAR              null,
   user_token           VARCHAR              null,
   role                 INT8                 null,
   grp                  INT8                 null,
   ip                   VARCHAR              null,
   port                 VARCHAR              null,
   auth_failed_count    INT4                 null,
   lock_duration        INT4                 null,
   visit_count          INT4                 null,
   attack_count         INT4                 null,
   lock_reason          VARCHAR              null,
   logon_time           INT8                 null,
   begin_lock_time      INT8                 null,
   creator              INT8                 null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   updated_by           INT8                 null,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   domain_id            INT8                 null,
   dynamic_attr         VARCHAR              null,
   addi                 JSONB                null,
   remark               VARCHAR              null,
   status               VARCHAR              null default '01',
   constraint PK_T_USER primary key (id)
);

comment on table t_user is
't_user';

comment on column t_user.id is
'用户内部编号';

comment on column t_user.external_id_type is
'用户外部标识类型';

comment on column t_user.external_id is
'用户外部标识';

comment on column t_user.category is
'用户分类';

comment on column t_user.type is
'用户类型, 
00:匿名用户, 0000-0001，未提供外部可识别标识用户，未付费，不可识别与联系
02:注册用户, 0000-0010，具备可识别信息
04:试用用户, 0000-0100，帐号有过期时间，使用了付费功能，具备可识别信息
08:机构上帝, 0000-1000，帐号有特定管理功能，具备可识别信息

10:测试用户, 0001-0000，用来测试的用户，具备可识别信息
80:系统上帝, 1000-0000，系统管理员功能，具备可识别信息
　　　　 ';

comment on column t_user.language is
'用户喜好语言';

comment on column t_user.country is
'国家';

comment on column t_user.province is
'省份';

comment on column t_user.city is
'城市';

comment on column t_user.addr is
'详细地址';

comment on column t_user.fuse_name is
'融合用户名称: coalesce( official_name,nickname,mobile_phone,email,account,u.id)';

comment on column t_user.official_name is
'姓名';

comment on column t_user.id_card_type is
'证件类型';

comment on column t_user.id_card_no is
'身份证号码';

comment on column t_user.mobile_phone is
'手机号码';

comment on column t_user.email is
'电子邮件';

comment on column t_user.account is
'登录账号';

comment on column t_user.gender is
'性别';

comment on column t_user.birthday is
'出生日期';

comment on column t_user.nickname is
'呢称';

comment on column t_user.avatar is
'头像';

comment on column t_user.avatar_type is
'头像类型, LINK: URL链接，B64: BASE64编码图片';

comment on column t_user.dev_id is
'终端设备标识,iOS/Android DeviceId';

comment on column t_user.dev_User_id is
'终端用户标识,google Account, iTunes Account';

comment on column t_user.dev_account is
'与设备关联的用于C2DM/APNS 的Android/iOS帐号';

comment on column t_user.cert is
'证书';

comment on column t_user.user_token is
'crypt(''pwd'',gen_salt(''bf''))';

comment on column t_user.role is
'最近用户使用角色编号';

comment on column t_user.grp is
'最近用户使用组编号';

comment on column t_user.ip is
'最近IP';

comment on column t_user.port is
'最近端口';

comment on column t_user.auth_failed_count is
'登录失败次数';

comment on column t_user.lock_duration is
'需要锁定时长，以秒计';

comment on column t_user.visit_count is
'访问计数';

comment on column t_user.attack_count is
'攻击次数';

comment on column t_user.lock_reason is
'锁定原因';

comment on column t_user.logon_time is
'最近登录时间';

comment on column t_user.begin_lock_time is
'开始锁定时间';

comment on column t_user.creator is
'创建者';

comment on column t_user.create_time is
'创建时间';

comment on column t_user.updated_by is
'更新者';

comment on column t_user.update_time is
'帐号信息更新时间';

comment on column t_user.domain_id is
'数据隶属';

comment on column t_user.dynamic_attr is
'动态属性，用于返回前端需要的基于计算的数据，表中无此数据，动态生成';

comment on column t_user.addi is
'用户定制数据';

comment on column t_user.remark is
'备注';

comment on column t_user.status is
'状态,00: 有效, 02: 禁止登录, 04: 锁定, 06: 攻击者, 08: 过期';

ALTER SEQUENCE t_user_id_seq RESTART WITH 20000;

delete from t_user where id <=20000;

insert into t_user(id,type,email,account,user_token,status,category) values
(1000,'80','kzz@gzhu.edu.cn','admin',crypt('cst4Ever',gen_salt('bf')),'00','sys^admin'),
(1002,'02','dawnfire@126.com','mickey',crypt('cst4Ever',gen_salt('bf')),'00','sys^admin'),
(1004,'04','kzz@tom.com','trialUser',crypt('cst4Ever',gen_salt('bf')),'00','sys^trial'),
(1008,'08','kmanager@gmail.com','organizationLeader',crypt('cst4Ever',gen_salt('bf')),'00','sys^admin'),
(1010,'10','kforce@gmail.com','tester',crypt('cst4Ever',gen_salt('bf')),'00','sys'),
(1110,'10','stu01@w2w.io','1110',crypt('1',gen_salt('bf')),'00','course^student'),
(1111,'10','stu02@w2w.io','1111',crypt('2',gen_salt('bf')),'00','course^student'),
(1212,'10','stu03@w2w.io','1212',crypt('3',gen_salt('bf')),'00','course^student'),
(1313,'10','stu04@w2w.io','1313',crypt('3',gen_salt('bf')),'00','course^student'),
(1314,'10','stu05@w2w.io','1314',crypt('5',gen_salt('bf')),'00','course^student'),

(1400,'10','course.admin@w2w.io','course.admin',crypt('0',gen_salt('bf')),'00','course^admin'),
(1402,'10','course.maintain@w2w.io','course.maintain',crypt('0',gen_salt('bf')),'00','course^maintain'),

(1404,'10','course.teacher1@w2w.io','t1',crypt('1',gen_salt('bf')),'00','course^teacher'),
(1406,'10','course.teacher2@w2w.io','t2',crypt('2',gen_salt('bf')),'00','course^teacher'),
(1408,'10','course.teacher3@w2w.io','t3',crypt('3',gen_salt('bf')),'00','course^teacher'),

(1410,'10','course.stu1@w2w.io','s1',crypt('1',gen_salt('bf')),'00','course^student'),
(1412,'10','course.stu2@w2w.io','s2',crypt('2',gen_salt('bf')),'00','course^student'),
(1414,'10','course.stu3@w2w.io','s3',crypt('3',gen_salt('bf')),'00','course^student'),
(1416,'10','course.stu4@w2w.io','s4',crypt('4',gen_salt('bf')),'00','course^student'),
(1418,'10','course.stu5@w2w.io','s5',crypt('5',gen_salt('bf')),'00','course^student'),

-- xkb default user

(10002,'02','admin@xkb888.cn','xkb_admin',crypt('cst4Ever',gen_salt('bf')),'00','xkb^admin'),
(10004,'04','sale@xkb888.cn','xkb_sale',crypt('cst4Ever',gen_salt('bf')),'00','xkb^sale'),
(10008,'08','school.admin@xkb888.cn','xkb_school_admin',crypt('cst4Ever',gen_salt('bf')),'00','xkb.school^admin'),
(10010,'10','school.statistics@xkb888.cn','xkb_school_statistics',crypt('cst4Ever',gen_salt('bf')),'00','xkb.school^statistics'),
(10050,'10','user1@xkb888.cn','10050',crypt('cst4Ever',gen_salt('bf')),'00','sys^user'),
(10052,'10','user2@xkb888.cn','10052',crypt('cst4Ever',gen_salt('bf')),'00','sys^user'),
(10054,'10','user3@xkb888.cn','10054',crypt('cst4Ever',gen_salt('bf')),'00','sys^user');


insert into t_user(id,account,category,id_card_type,official_name,id_card_no,user_token) values
(10100,'10001','xkb.user','身份证','李玟筱','430523200011094320',crypt('cst4Ever',gen_salt('bf'))),
(10102,'10002','xkb.user','身份证','徐夏龙','430223200108148018',crypt('cst4Ever',gen_salt('bf'))),
(10104,'10004','xkb.user','身份证','刘思婕','430225200203308545',crypt('cst4Ever',gen_salt('bf'))),
(10106,'10006','xkb.user','身份证','杨世杰','430223200111013210',crypt('cst4Ever',gen_salt('bf'))),
(10108,'10008','xkb.user','身份证','吴博宇','43021120011022001x',crypt('cst4Ever',gen_salt('bf'))),
(10110,'10010','xkb.user','身份证','郑子龙','330322200106121618',crypt('cst4Ever',gen_salt('bf'))),
(10112,'10012','xkb.user','身份证','蒋志','430523200106098017',crypt('cst4Ever',gen_salt('bf'))),
(10114,'10014','xkb.user','身份证','刘洁琼','430223200108097249',crypt('cst4Ever',gen_salt('bf'))),
(10116,'10016','xkb.user','身份证','陈嘉正','430203200107230217',crypt('cst4Ever',gen_salt('bf'))),
(10118,'10018','xkb.user','身份证','向菁','431224200104200031',crypt('cst4Ever',gen_salt('bf')));

insert into t_user(id,account,category,official_name,mobile_phone,user_token) values   
(10120,'10020','xkb.user','卢星宇','13342885601',crypt('cst4Ever',gen_salt('bf'))),
(10122,'10022','xkb.user','李嘉文','13342885602',crypt('cst4Ever',gen_salt('bf'))),
(10124,'10024','xkb.user','罗晟宇','13342885603',crypt('cst4Ever',gen_salt('bf'))),
(10126,'10026','xkb.user','戴仟仟','13342885604',crypt('cst4Ever',gen_salt('bf'))),
(10128,'10028','xkb.user','姜湘晨','13342885605',crypt('cst4Ever',gen_salt('bf'))),
(10130,'10030','xkb.user','周方圆','13342885606',crypt('cst4Ever',gen_salt('bf'))),
(10132,'10032','xkb.user','宓楚钰','13342885607',crypt('cst4Ever',gen_salt('bf'))),
(10134,'10034','xkb.user','李文琪','13342885608',crypt('cst4Ever',gen_salt('bf'))),
(10136,'10036','xkb.user','刘旭兵','13342885609',crypt('cst4Ever',gen_salt('bf'))),
(10138,'10038','xkb.user','康朝岳','13342885600',crypt('cst4Ever',gen_salt('bf')));


insert into t_user(id,type,email,account,user_token,status,category) values
(10140,'02','admin10040@xkb888.cn','xkb_admin_10040',crypt('cst4Ever',gen_salt('bf')),'00','xkb^admin'),
(10142,'02','admin10042@xkb888.cn','xkb_admin_10042',crypt('cst4Ever',gen_salt('bf')),'00','xkb^admin'),
(10144,'02','admin10044@xkb888.cn','xkb_admin_10044',crypt('cst4Ever',gen_salt('bf')),'00','xkb^admin');


insert into t_user(id,email,account,user_token,status,category) values
(10156,'user4@xkb888.cn','10056',crypt('cst4Ever',gen_salt('bf')),'00','sys^user'),
(10158,'user5@xkb888.cn','10058',crypt('cst4Ever',gen_salt('bf')),'00','sys^user'),
(10160,'user6@xkb888.cn','10060',crypt('cst4Ever',gen_salt('bf')),'00','sys^user'),
(10162,'user7@xkb888.cn','10062',crypt('cst4Ever',gen_salt('bf')),'00','sys^user'),
(10164,'user8@xkb888.cn','10064',crypt('cst4Ever',gen_salt('bf')),'00','sys^user'),
(10168,'user9@xkb888.cn','10068',crypt('cst4Ever',gen_salt('bf')),'00','sys^user'),
(10170,'user10@xkb888.cn','10070',crypt('cst4Ever',gen_salt('bf')),'00','sys^user'),
(10172,'user11@xkb888.cn','10072',crypt('cst4Ever',gen_salt('bf')),'00','sys^user'),
(10174,'user12@xkb888.cn','10074',crypt('cst4Ever',gen_salt('bf')),'00','sys^user'),
(10176,'user13@xkb888.cn','10076',crypt('cst4Ever',gen_salt('bf')),'00','sys^user'),
(10178,'user14@xkb888.cn','10078',crypt('cst4Ever',gen_salt('bf')),'00','sys^user'),
(10180,'user15@xkb888.cn','10080',crypt('cst4Ever',gen_salt('bf')),'00','sys^user');






update t_user set role=566;

update t_user set user_token=crypt('x2K3c',gen_salt('bf')) where id in(10002,10004,10008,10010,10040,10042,10044);

/*==============================================================*/
/* Index: idx_user_account                                      */
/*==============================================================*/
create unique index if not exists  idx_user_account on t_user (
account
);

/*==============================================================*/
/* Index: idx_user_externalID                                   */
/*==============================================================*/
create unique index if not exists  idx_user_externalID on t_user (
external_id,
external_id_type
);

/*==============================================================*/
/* Index: idx_user_nickName                                     */
/*==============================================================*/
create  index if not exists  idx_user_nickName on t_user (
nickname
);

/*==============================================================*/
/* Index: idx_user_phone                                        */
/*==============================================================*/
create unique index if not exists  idx_user_phone on t_user (
mobile_phone
);

/*==============================================================*/
/* Index: idx_user_email                                        */
/*==============================================================*/
create unique index if not exists  idx_user_email on t_user (
email
);

/*==============================================================*/
/* Index: idx_user_id_card_no                                   */
/*==============================================================*/
create unique index if not exists  idx_user_id_card_no on t_user (
id_card_type,
id_card_no
);

/*==============================================================*/
/* Index: idx_user_offcial_name                                 */
/*==============================================================*/
create  index if not exists  idx_user_offcial_name on t_user (
official_name
);

/*==============================================================*/
/* Table: t_domain_asset                                        */
/*==============================================================*/
create table if not exists  t_domain_asset (
   id                   SERIAL not null,
   r_type               VARCHAR              not null,
   domain_id            INT8                 not null,
   asset_id             INT8                 not null,
   id_on_domain         VARCHAR              null,
   grant_source         VARCHAR              null,
   data_access_mode     VARCHAR              null,
   data_scope           JSONB                null,
   updated_by           INT8                 null,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   creator              INT8                 null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   addi                 JSONB                null,
   remark               VARCHAR              null,
   status               VARCHAR              null default '01',
   constraint PK_T_DOMAIN_ASSET primary key (id)
);

comment on table t_domain_asset is
'define

user domain relation
domain api relation
other relation';

comment on column t_domain_asset.id is
'id';

comment on column t_domain_asset.r_type is
'关系类型, ud: user of domain, da: API of domain';

comment on column t_domain_asset.domain_id is
'对象归属域编号';

comment on column t_domain_asset.asset_id is
'对象编号, 如账号、API接口';

comment on column t_domain_asset.id_on_domain is
'仅当r_type=''ud''时有效，基于用户域的用户编码，如广州大学员工号，后勤部员工号，采购组采购员编号，保卫科保安员工号';

comment on column t_domain_asset.grant_source is
'grant:数据权限由t_relation中left_type:t_domain.id与left_type:t_user.id获得的数据决定,或data_scope中数据决定，但data_scope与t_relation只能存在一种，如果data_scope有效，则忽略t_relation;

cousin:忽略data_scope与t_relation, 授权数据由被过虑数据的domain_id决定,即被过虑数据的domain_id 与登录用户的t_user.domain_id相同或级别更低的数据，例如
    用户的t_user.domain为xkb^admin而数据的domain为xkb.school^admin，则用户可以获得该数据

self: 被过虑数据的creator 与登录用户的t_user.id相同

api: 由功能(api)自己决定 ';

comment on column t_domain_asset.data_access_mode is
'数据访问类型, full:可读写, read: 只读, write: 写, partial: 部分写/混合';

comment on column t_domain_asset.data_scope is
'当grant_source是grant时,以json数据方式提供数据授权范围格式为:
  {"granter":"t_user.id","grantee":"t_school.id","data":[1234,456,789]}
granter: 代表数据拥有者, t_user.id代表用户, t_domain.id代表角色,t_api.id代表功能
grantee: 代表拥有的数据,t_school.id代表可以访问的机构列表。
授权数据如果存储在t_relation中则各项分别对应如下
granter对应left_type, left_key对应t_user_domain.sys_user或t_domain_api.domain
grantee对应right_type, right_key对应right_type的意义';

comment on column t_domain_asset.updated_by is
'更新者';

comment on column t_domain_asset.update_time is
'帐号信息更新时间';

comment on column t_domain_asset.creator is
'本数据创建者';

comment on column t_domain_asset.create_time is
'生成时间';

comment on column t_domain_asset.addi is
'附加信息';

comment on column t_domain_asset.remark is
'备注';

comment on column t_domain_asset.status is
'状态，00：草稿，01：有效，02：作废';

with cte as ( select id,domain,name from t_domain )
insert into t_domain_asset(r_type, domain_id,asset_id)
select 'ud',cte.id,u.id
from t_user u join cte on u.category=cte.domain
on conflict do nothing;

/*==============================================================*/
/* Index: domain_asset_relation                                 */
/*==============================================================*/
create unique index if not exists  domain_asset_relation on t_domain_asset (
r_type,
domain_id,
asset_id
);

/*==============================================================*/
/* Index: app_user_id_idx                                       */
/*==============================================================*/
create unique index if not exists  app_user_id_idx on t_domain_asset (
r_type,
domain_id,
asset_id,
id_on_domain
);

/*==============================================================*/
/* Table: t_exam_info                                           */
/*==============================================================*/
create table if not exists  t_exam_info (
   id                   SERIAL               not null,
   name                 VARCHAR(150)         null default '未命名考试',
   rules                TEXT                 null,
   type                 VARCHAR(50)          null,
   mode                 VARCHAR(50)          null,
   files                JSONB                null,
   submitted            BOOL                 null,
   creator              INT8                 not null,
   create_time          INT8                 null,
   updated_by           INT8                 null,
   update_time          INT8                 null,
   status               VARCHAR(150)         null,
   addi                 JSONB                null,
   exam_room_ids        bigint[]             null,
   exam_room_invigilator_count JSONB                null,
   domain_id            INT8                 null,
   constraint PK_T_EXAM_INFO primary key (id)
);

comment on table t_exam_info is
'考试信息表';

comment on column t_exam_info.id is
'考试编号';

comment on column t_exam_info.name is
'考试名称';

comment on column t_exam_info.rules is
'考试规则';

comment on column t_exam_info.type is
'考试类型 00：平时考试 02：期末成绩考试  04：资格证考试';

comment on column t_exam_info.mode is
'考试方式 00：线上考试  02：线下考试';

comment on column t_exam_info.files is
'考试附件资料';

comment on column t_exam_info.submitted is
'考试成绩是否已提交';

comment on column t_exam_info.creator is
'创建者';

comment on column t_exam_info.create_time is
'创建时间';

comment on column t_exam_info.updated_by is
'更新者';

comment on column t_exam_info.update_time is
'更新时间';

comment on column t_exam_info.status is
'状态  00：未发布 02：待开始  04：进行中 06：已结束 08：已归档 10：考试异常 12：已删除
14：临时  16：已作废';

comment on column t_exam_info.addi is
'附加信息';

comment on column t_exam_info.exam_room_ids is
'考场id数组';

comment on column t_exam_info.exam_room_invigilator_count is
'记录当前每个考场的所需监考员数量，例：[{exam_room_id:1,invigilator_count:1}]';

comment on column t_exam_info.domain_id is
'domain_id';

/*==============================================================*/
/* Table: t_exam_paper                                          */
/*==============================================================*/
create table if not exists  t_exam_paper (
   id                   SERIAL               not null,
   exam_session_id      INT8                 null,
   practice_id          INT8                 null,
   name                 VARCHAR(256)         null,
   creator              INT8                 not null,
   create_time          INT8                 null,
   updated_by           INT8                 null,
   update_time          INT8                 null,
   addi                 JSONB                null,
   status               VARCHAR(10)          null default '00',
   constraint PK_T_EXAM_PAPER primary key (id)
);

comment on table t_exam_paper is
'考卷';

comment on column t_exam_paper.id is
'考卷ID';

comment on column t_exam_paper.exam_session_id is
'考试场次ID，标识考卷用于哪一场考试场次';

comment on column t_exam_paper.practice_id is
'练习ID，标识考卷用于哪一个练习';

comment on column t_exam_paper.name is
'考卷名称';

comment on column t_exam_paper.creator is
'创建者';

comment on column t_exam_paper.create_time is
'创建时间';

comment on column t_exam_paper.updated_by is
'更新者';

comment on column t_exam_paper.update_time is
'更新时间';

comment on column t_exam_paper.addi is
'附加信息';

comment on column t_exam_paper.status is
'状态 00：使用中，02：归档，04：废弃';

/*==============================================================*/
/* Table: t_exam_paper_group                                    */
/*==============================================================*/
create table if not exists  t_exam_paper_group (
   id                   INT4                 not null,
   exam_paper_id        INT8                 not null,
   name                 TEXT                 null,
   "order"              INT4                 null,
   creator              INT8                 not null,
   create_time          INT8                 null,
   updated_by           INT8                 null,
   update_time          INT8                 null,
   addi                 JSONB                null,
   status               VARCHAR(10)          null default '00',
   constraint PK_T_EXAM_PAPER_GROUP primary key (id)
);

comment on table t_exam_paper_group is
'学生考卷题组';

comment on column t_exam_paper_group.id is
'答卷题组id';

comment on column t_exam_paper_group.exam_paper_id is
'学生答卷ID';

comment on column t_exam_paper_group.name is
'题组名称';

comment on column t_exam_paper_group."order" is
'题组排序';

comment on column t_exam_paper_group.creator is
'创建者';

comment on column t_exam_paper_group.create_time is
'创建时间';

comment on column t_exam_paper_group.updated_by is
'更新者';

comment on column t_exam_paper_group.update_time is
'更新时间';

comment on column t_exam_paper_group.addi is
'附加信息';

comment on column t_exam_paper_group.status is
'状态 00：正常， 02：异常';

/*==============================================================*/
/* Table: t_exam_paper_question                                 */
/*==============================================================*/
create table if not exists  t_exam_paper_question (
   id                   SERIAL               not null,
   group_id             INT8                 null,
   score                FLOAT8               null,
   "order"              INT4                 null,
   type                 VARCHAR(128)         null,
   content              TEXT                 null,
   options              JSONB                null,
   answers              JSONB                null,
   analysis             TEXT                 null,
   title                TEXT                 null,
   answer_file_path     JSONB                null,
   test_file_path       JSONB                null,
   input                VARCHAR(255)         null,
   output               VARCHAR(255)         null,
   example              JSONB                null,
   repo                 JSONB                null,
   commit_id            JSONB                null,
   creator              INT8                 not null,
   create_time          INT8                 null,
   updated_by           INT8                 null,
   update_time          INT8                 null,
   addi                 JSONB                null,
   status               VARCHAR(10)          null default '00',
   question_attachments_path JSONB                null,
   constraint PK_T_EXAM_PAPER_QUESTION primary key (id)
);

comment on table t_exam_paper_question is
'考卷题目';

comment on column t_exam_paper_question.id is
'考卷题目ID';

comment on column t_exam_paper_question.group_id is
'所属题组ID';

comment on column t_exam_paper_question.score is
'题目分值';

comment on column t_exam_paper_question."order" is
'原始考卷的题目初始顺序';

comment on column t_exam_paper_question.type is
'题目类型 "00":单选题 "02":多选题 "04": 判断题 "06": 填空题 "08": 简答题 "10": 编程题';

comment on column t_exam_paper_question.content is
'理论题目内容';

comment on column t_exam_paper_question.options is
'理论题目选项填空项';

comment on column t_exam_paper_question.answers is
'理论题目答案';

comment on column t_exam_paper_question.analysis is
'理论题目解析';

comment on column t_exam_paper_question.title is
'编程题目题干';

comment on column t_exam_paper_question.answer_file_path is
'编程题目答案文件路径';

comment on column t_exam_paper_question.test_file_path is
'编程题目测试文件路径';

comment on column t_exam_paper_question.input is
'编程题目输入';

comment on column t_exam_paper_question.output is
'编程题目输出';

comment on column t_exam_paper_question.example is
'编程题目示例';

comment on column t_exam_paper_question.repo is
'仓库';

comment on column t_exam_paper_question.commit_id is
'编程题提交ID，用来当版本号';

comment on column t_exam_paper_question.creator is
'创建者';

comment on column t_exam_paper_question.create_time is
'创建时间';

comment on column t_exam_paper_question.updated_by is
'更新者';

comment on column t_exam_paper_question.update_time is
'更新时间';

comment on column t_exam_paper_question.addi is
'附加信息';

comment on column t_exam_paper_question.status is
'状态 ：00：使用中，02：归档，04：废弃';

comment on column t_exam_paper_question.question_attachments_path is
'题目附件url数组';

/*==============================================================*/
/* Table: t_exam_record                                         */
/*==============================================================*/
create table if not exists  t_exam_record (
   id                   INT4                 not null,
   exam_room            int8                 not null,
   exam_session         int8                 not null,
   content              varchar(5000)        null,
   basic_eval           VARCHAR(150)         null,
   creator              INT8                 not null,
   create_time          timestamp            null,
   updated_by           INT8                 null,
   update_time          timestamp            null,
   addi                 jsonb                null,
   status               varchar(150)         null,
   constraint PK_T_EXAM_RECORD primary key (id)
);

comment on table t_exam_record is
'考场记录表';

comment on column t_exam_record.id is
'记录ID';

comment on column t_exam_record.exam_room is
'考场ID';

comment on column t_exam_record.exam_session is
'考试场次ID';

comment on column t_exam_record.content is
'记录内容';

comment on column t_exam_record.basic_eval is
'基本情况评估 00: 良好 02: 一般 04: 较差';

comment on column t_exam_record.creator is
'创建者';

comment on column t_exam_record.create_time is
'创建时间';

comment on column t_exam_record.updated_by is
'最近一次的更新者';

comment on column t_exam_record.update_time is
'最近一次更新的时间';

comment on column t_exam_record.addi is
'附加信息';

comment on column t_exam_record.status is
'状态码 00:正常 02:失效(删除)';

/*==============================================================*/
/* Table: t_exam_room                                           */
/*==============================================================*/
create table if not exists  t_exam_room (
   id                   SERIAL               not null,
   exam_site            INT8                 not null,
   name                 VARCHAR(150)         null,
   capacity             INT4                 null,
   creator              INT8                 not null,
   create_time          TIMESTAMP            null,
   updated_by           INT8                 null,
   update_time          TIMESTAMP            null,
   status               VARCHAR(150)         null default '0',
   addi                 JSONB                null,
   constraint PK_T_EXAM_ROOM primary key (id)
);

comment on table t_exam_room is
'考场表';

comment on column t_exam_room.id is
'考场编号';

comment on column t_exam_room.exam_site is
'考点编号';

comment on column t_exam_room.name is
'考场名字';

comment on column t_exam_room.capacity is
'考场容量';

comment on column t_exam_room.creator is
'创建者';

comment on column t_exam_room.create_time is
'创建时间';

comment on column t_exam_room.updated_by is
'更新者';

comment on column t_exam_room.update_time is
'更新时间';

comment on column t_exam_room.status is
'考场状态 00：正常 02：故障 04：占用 06：已删除';

comment on column t_exam_room.addi is
'附加信息';

/*==============================================================*/
/* Table: t_exam_session                                        */
/*==============================================================*/
create table if not exists  t_exam_session (
   id                   SERIAL               not null,
   exam_id              INT8                 not null,
   paper_id             INT8                 not null,
   session_num          INT8                 null,
   mark_method          VARCHAR(50)          not null,
   period_mode          VARCHAR(50)          null,
   start_time           INT8                 null,
   end_time             INT8                 null,
   duration             INT4                 null,
   question_shuffled_mode VARCHAR(50)          null,
   mark_mode            VARCHAR(100)         null,
   name_visibility_in   BOOL                 null,
   creator              INT8                 not null,
   create_time          INT8                 null,
   updated_by           INT8                 null,
   update_time          INT8                 null,
   status               VARCHAR(150)         null,
   addi                 JSONB                null,
   late_entry_time      INT8                 null,
   early_submission_time INT8                 null,
   reviewer_ids         bigint[]             null,
   basic_eval           VARCHAR(150)         null,
   record               VARCHAR(1500)        null,
   constraint PK_T_EXAM_SESSION primary key (id)
);

comment on table t_exam_session is
'考试场次表';

comment on column t_exam_session.id is
'编号';

comment on column t_exam_session.exam_id is
'考试编号';

comment on column t_exam_session.paper_id is
'试卷编号';

comment on column t_exam_session.session_num is
'考试场次序号标识（试卷1,试卷2...)';

comment on column t_exam_session.mark_method is
'批卷方式 00：人工批卷 02：自动批卷 04：ai批卷';

comment on column t_exam_session.period_mode is
'考试时段模式 00：固定时段考试 02：灵活时段考试';

comment on column t_exam_session.start_time is
'考试开始时间';

comment on column t_exam_session.end_time is
'考试结束时间';

comment on column t_exam_session.duration is
'考试时长';

comment on column t_exam_session.question_shuffled_mode is
'乱序方式 00：既有试题乱序也有选项乱序 02：选项乱序 04：试题乱序 06：都不选择';

comment on column t_exam_session.mark_mode is
'批改配置，包括批卷模式 00：不需要批改  02：全卷多评 04：试卷分配 06：题组专评 08：逐卷批改 10：逐题批改';

comment on column t_exam_session.name_visibility_in is
'当需要人工批卷时，是否需要在批改中显示学生姓名';

comment on column t_exam_session.creator is
'创建者';

comment on column t_exam_session.create_time is
'创建时间';

comment on column t_exam_session.updated_by is
'更新者';

comment on column t_exam_session.update_time is
'更新时间';

comment on column t_exam_session.status is
'状态 00：未发布 02：待开始  04：进行中 06：已结束  08：批改中 10：已批改 12：已提交 14：已删除 16：已作废';

comment on column t_exam_session.addi is
'附加信息';

comment on column t_exam_session.late_entry_time is
'考试开始后最晚能进入考场的时间，如考试开始后30分钟内可进入考场';

comment on column t_exam_session.early_submission_time is
'考试结束前x分钟可交卷';

comment on column t_exam_session.reviewer_ids is
'批阅员ID数组';

comment on column t_exam_session.basic_eval is
'基本情况评估 00: 良好 02: 一般 04: 较差';

comment on column t_exam_session.record is
'考试场次记录，可用于保存本考试场次中的异常情况记录';

/*==============================================================*/
/* Table: t_exam_site                                           */
/*==============================================================*/
create table if not exists  t_exam_site (
   id                   SERIAL               not null,
   name                 VARCHAR(150)         null,
   address              VARCHAR(100)         null,
   server_host          VARCHAR(100)         null,
   creator              INT8                 not null,
   create_time          TIMESTAMP            null,
   updated_by           INT8                 null,
   update_time          TIMESTAMP            null,
   status               VARCHAR(150)         null default '0',
   addi                 JSONB                null,
   admin                INT8                 null,
   sys_user             INT8                 null,
   domain_id            INT8                 null,
   constraint PK_T_EXAM_SITE primary key (id)
);

comment on table t_exam_site is
'考点表';

comment on column t_exam_site.id is
'考点编号';

comment on column t_exam_site.name is
'考点名字';

comment on column t_exam_site.address is
'考点地址';

comment on column t_exam_site.server_host is
'考点服务器地址(包含端口)';

comment on column t_exam_site.creator is
'创建者';

comment on column t_exam_site.create_time is
'创建时间';

comment on column t_exam_site.updated_by is
'更新者';

comment on column t_exam_site.update_time is
'更新时间';

comment on column t_exam_site.status is
'考点状态 00：空闲 02：故障 04：删除';

comment on column t_exam_site.addi is
'附加信息';

comment on column t_exam_site.admin is
'考点负责人';

comment on column t_exam_site.sys_user is
'考点服务器系统账号ID';

comment on column t_exam_site.domain_id is
'数据所属域';

/*==============================================================*/
/* Table: t_examinee                                            */
/*==============================================================*/
create table if not exists  t_examinee (
   id                   SERIAL               not null,
   student_id           INT8                 null,
   serial_number        INT4                 null,
   exam_room            INT8                 null,
   exam_session_id      INT8                 null,
   examinee_number      character varying    null,
   start_time           INT8                 null,
   end_time             INT8                 null,
   remark               VARCHAR              null,
   creator              INT8                 not null,
   create_time          INT8                 null,
   updated_by           INT8                 null,
   update_time          INT8                 null,
   status               VARCHAR(150)         null,
   addi                 JSONB                null,
   extra_time           bigint               null default '0',
   exam_paper_id        bigint               null,
   constraint PK_T_EXAMINEE primary key (id)
);

comment on table t_examinee is
'考生表';

comment on column t_examinee.id is
'id';

comment on column t_examinee.student_id is
'学生ID';

comment on column t_examinee.serial_number is
'考生序号';

comment on column t_examinee.exam_room is
'考场信息';

comment on column t_examinee.exam_session_id is
'考试场次编号';

comment on column t_examinee.examinee_number is
'考生准考证号';

comment on column t_examinee.start_time is
'start_time';

comment on column t_examinee.end_time is
'end_time';

comment on column t_examinee.remark is
'remark';

comment on column t_examinee.creator is
'创建者';

comment on column t_examinee.create_time is
'创建时间';

comment on column t_examinee.updated_by is
'更新者';

comment on column t_examinee.update_time is
'更新时间';

comment on column t_examinee.status is
'状态 00：正常考 02：缺考 04：补考 06：作弊 08：已删除 10：已交卷 12：待同步 14：考试异常 16：已作废';

comment on column t_examinee.addi is
'附加信息(备注),可以保存考试异常的原因等';

comment on column t_examinee.extra_time is
'考试延长时间(毫秒).考试实际结束时间=当前考试结束时间+延长时间';

comment on column t_examinee.exam_paper_id is
'考卷ID';

/*==============================================================*/
/* Table: t_expertise                                           */
/*==============================================================*/
create table if not exists  t_expertise (
   id                   SERIAL not null,
   belongto             INT8                 null,
   name                 VARCHAR              null,
   limn                 VARCHAR              null,
   creator              INT8                 null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   status               VARCHAR              null,
   constraint PK_T_EXPERTISE primary key (id)
);

comment on table t_expertise is
'知识能力领域表';

comment on column t_expertise.id is
'编号';

comment on column t_expertise.belongto is
'上级expertise';

comment on column t_expertise.name is
'知识能力领域名称';

comment on column t_expertise.limn is
'描述';

comment on column t_expertise.creator is
'创建者';

comment on column t_expertise.create_time is
'创建时间';

comment on column t_expertise.update_time is
'更新时间';

comment on column t_expertise.status is
'可用，禁用';

ALTER SEQUENCE t_expertise_id_seq RESTART WITH 10000;

/*==============================================================*/
/* Table: t_external_domain_conf                                */
/*==============================================================*/
create table if not exists  t_external_domain_conf (
   id                   SERIAL not null,
   app_id               VARCHAR              not null,
   app_type             VARCHAR              not null default 'wx_mp',
   app_name             VARCHAR              not null,
   tokens               JSONB                not null,
   updated_by           INT8                 null,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   creator              INT8                 null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   domain_id            INT8                 null,
   addi                 JSONB                null,
   remark               VARCHAR              null,
   status               VARCHAR              null default '02',
   constraint PK_T_EXTERNAL_DOMAIN_CONF primary key (id),
   constraint AK_EXT_D_C_APP_ID_T_EXTERN unique (app_id),
   constraint AK_EXT_D_C_APP_NAME_T_EXTERN unique (app_name)
);

comment on table t_external_domain_conf is
'外部系统访问标识';

comment on column t_external_domain_conf.id is
'编号';

comment on column t_external_domain_conf.app_id is
'外部应用标识，如微信公众号appID';

comment on column t_external_domain_conf.app_type is
'wx_mp: 微信公众号, wx_open: 微信开放平台, ali: 阿里, ui: 联保';

comment on column t_external_domain_conf.app_name is
'应用名称，如校快保2019，近邻科技';

comment on column t_external_domain_conf.tokens is
'例如，联保：{"appID":"xkbtest",	"appSecret":"123456",
	"branchID":"ba331eb1851d4d8bb5e838dfbf9e09d7",
	"userID":"e1d9441f16284d99a8c2732aedca5753"
}';

comment on column t_external_domain_conf.updated_by is
'更新者';

comment on column t_external_domain_conf.update_time is
'帐号信息更新时间';

comment on column t_external_domain_conf.creator is
'本数据创建者';

comment on column t_external_domain_conf.create_time is
'生成时间';

comment on column t_external_domain_conf.domain_id is
'数据隶属';

comment on column t_external_domain_conf.addi is
'附加信息';

comment on column t_external_domain_conf.remark is
'备注';

comment on column t_external_domain_conf.status is
'状态，00：草稿，02：有效，04: 停用，06：作废';

ALTER SEQUENCE t_external_domain_conf_id_seq RESTART WITH 20000;
insert into t_external_domain_conf (app_type,creator, domain_id,app_id, app_name, tokens) values
('ui',1000,177,'xkbtest','联保','{
    "appID": "xkbtest",
    "appSecret": "123456",
    "branchID": "ba331eb1851d4d8bb5e838dfbf9e09d7",
    "getTokenURL": "http://39.108.238.42:8090/oauth2/authorize",
    "policy": "http://39.108.238.42:8090/api/picc/xkb/policy",
    "policyCancel": "http://39.108.238.42:8090/api/picc/xkb/cannelPolicy",
    "policyDetail": "http://39.108.238.42:8090/api/picc/xkb/policyDetail",
    "productCode": "PICCSHOOL_TEST",
    "refreshTokenURL": "http://39.108.238.42:8090/oauth2/token_refresh",
    "userID": "e1d9441f16284d99a8c2732aedca5753"
}'),('wx_mp',1000,177,'wx0fefb244eeef3422','广州近邻微信公众号','{
    "dstURL": "https://qnear.cn/api/wxLogin?role=anonymous",
    "mustGetWxMpAcsToken": false,
    "realIPServ": "http://qnear.cn:64000/gipQry?q=142857",
    "verifyBussinessDomainFileContent": "NoNLb44EuoLJ7ybT",
    "verifyBussinessDomainFileName": "/MP_verify_NoNLb44EuoLJ7ybT.txt",
    "服务号消息接口": "---------------------",
    "wxServeURI": "/api/wxServe",
    "微信公众号": "--------------------------",
    "wxMpAppID": "wx0fefb244eeef3422",
    "wxMpAppSecret": "9e4a4dfd0da11c0cb473c08b758a2582",
    "wxMpEncodingAESKey": "GudE8hGPv2Ujzf9UlO1W7oNgJcfOlmKlGC8NmiKX6WW",
    "wxMpServeToken": "GynostemmaPentaphylla",
    "微信开放平台": "--------------------------------",
    "wxOpenAppID": "wxbbcdc7faf43cecec",
    "wxOpenAppSecret": "f4ce6525dbfd7e53376cc15dea624ce8",
    "wxOrderApiV3Cert": "z4fo7AEDLVdshbGWvTnNxOJvtI3nH8yr",
    "微信支付": "-------------------------",
    "wxMCHID": "1538924421"
}'),('wx_mp',1000,177,'wx9bf2de6adcc2a356','泰合微信公众号','{
    "dstURL": "https://be.xkb888.cn/api/wxLogin?role=anonymous",
    "mustGetWxMpAcsToken": false,
    "realIPServ": "http://qnear.cn:64000/gipQry?q=142857",
    "verifyBussinessDomainFileContent": "NoNLb44EuoLJ7ybT",
    "verifyBussinessDomainFileName": "/MP_verify_NoNLb44EuoLJ7ybT.txt",
    "wxMCHID": "1524282551",
    "wxMpAppID": "wx9bf2de6adcc2a356",
    "wxMpAppSecret": "76c95bf4bb325d17e6bb7c1f1e8bcc25",
    "wxOpenAppID": "wxbbcdc7faf43cecec",
    "wxOpenAppSecret": "f4ce6525dbfd7e53376cc15dea624ce8",
    "wxOrderApiV3Cert": "123456789qwertyuiopasdfghjklzxcv",
    "wxServeURI": "/api/wxServe",
    "xkbAdminURL": "https://qnear.cn/xkb/bg"
}'),('wx_mp',1000,177,'wx0744ddb3680af80e','校快保2019微信公众号','{
		"dstURL": "https://be.xkb888.cn/api/wxLogin?role=anonymous",
    "mustGetWxMpAcsToken": false,
    "realIPServ": "http://qnear.cn:64000/gipQry?q=142857",
    "verifyBussinessDomainFileContent": "NoNLb44EuoLJ7ybT",
    "verifyBussinessDomainFileName": "/MP_verify_NoNLb44EuoLJ7ybT.txt",
    "wxMCHID": "1524282551",
    "wxMpAppID": "wx0744ddb3680af80e",
    "wxMpAppSecret": "e7c3b5e5d2595cc0c0688617f7b90b9e",
    "wxMpEncodingAESKey": "gayDPA96bH7UAB6Gyd7phupdVzUUgB89P9AYU67U70v",
    "wxMpServeToken": "mmSRC1NMR4FozfGMMz8PvRBmPsm616co",
    "wxOpenAppID": "wxbbcdc7faf43cecec",
    "wxOpenAppSecret": "f4ce6525dbfd7e53376cc15dea624ce8",
    "wxOrderApiV3Cert": "123456789qwertyuiopasdfghjklzxcv",
    "wxServeURI": "/api/wxServe",
    "xkbAdminURL": "https://qnear.cn/xkb/bg"
	}'),('wx_open',1000,177,'wxbbcdc7faf43cecec','近邻微信开放平台','{
    "mustGetWxMpAcsToken": false,
    "verifyBussinessDomainFileContent": "NoNLb44EuoLJ7ybT",
    "verifyBussinessDomainFileName": "/MP_verify_NoNLb44EuoLJ7ybT.txt",
    "wxOpenAppID": "wxbbcdc7faf43cecec",
    "wxOpenAppSecret": "f4ce6525dbfd7e53376cc15dea624ce8"
	}'),('wx_open',1000,177,'wx3e95b4d09d0aeda2','XKB微信开放平台','{
    "mustGetWxMpAcsToken": false,
    "verifyBussinessDomainFileContent": "NoNLb44EuoLJ7ybT",
    "verifyBussinessDomainFileName": "/MP_verify_NoNLb44EuoLJ7ybT.txt",
    "wxOpenAppID": "wx3e95b4d09d0aeda2",
    "wxOpenAppSecret": "1d2312561a5dade28a601a6d65d9aa4a"
	}');
    
    update t_external_domain_conf set status='04' where app_name='泰合微信公众号';

/*==============================================================*/
/* Table: t_external_domain_user                                */
/*==============================================================*/
create table if not exists  t_external_domain_user (
   id                   SERIAL not null,
   user_id              INT8                 not null,
   business_domain_id   VARCHAR              not null,
   user_domain_id       VARCHAR              not null,
   user_domain_union_id VARCHAR              null,
   apply_to             VARCHAR              null,
   domain_type          VARCHAR              not null,
   creator              INT8                 null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   domain_id            INT8                 null,
   addi                 JSONB                null,
   status               VARCHAR              null default '02',
   constraint PK_T_EXTERNAL_DOMAIN_USER primary key (id)
);

comment on table t_external_domain_user is
'第三方平台用户标识';

comment on column t_external_domain_user.id is
'编号';

comment on column t_external_domain_user.user_id is
'系统用户编号';

comment on column t_external_domain_user.business_domain_id is
'业务域甲方，如微信公众号appID，广州大学财务系统';

comment on column t_external_domain_user.user_domain_id is
'业务域乙方，如微信公众号openID，广州大学财务系统用户账号';

comment on column t_external_domain_user.user_domain_union_id is
'业务域乙方唯一标识，如微信unionID，广州大学教工编号/学号';

comment on column t_external_domain_user.apply_to is
'该ID用途，如用于支付，标识用户';

comment on column t_external_domain_user.domain_type is
'wx_mp: 微信公众号, wx_open: 微信开放平台, ali: 阿里, ui: 联保';

comment on column t_external_domain_user.creator is
'本数据创建者';

comment on column t_external_domain_user.create_time is
'生成时间';

comment on column t_external_domain_user.domain_id is
'数据隶属';

comment on column t_external_domain_user.addi is
'附加信息';

comment on column t_external_domain_user.status is
'状态，00：草稿，02：有效，04：禁用，06：作废';

ALTER SEQUENCE t_external_domain_user_id_seq RESTART WITH 20000;

/*==============================================================*/
/* Index: idx_ext_user_domain_b_d_u                             */
/*==============================================================*/
create unique index if not exists  idx_ext_user_domain_b_d_u on t_external_domain_user (
business_domain_id,
user_domain_id,
domain_type
);

/*==============================================================*/
/* Table: t_file                                                */
/*==============================================================*/
create table if not exists  t_file (
   id                   SERIAL not null,
   file_oid             OID                  null,
   file_name            VARCHAR              not null,
   path                 VARCHAR              not null,
   belongto_path        VARCHAR              not null,
   digest               VARCHAR              not null,
   size                 INT8                 null,
   create_time          INT8                 not null default (extract(epoch from current_timestamp)*1000)::bigint,
   creator              INT8                 not null,
   domain_id            INT8                 null,
   count                INT4                 null default 1,
   belongto             INT8                 null,
   limn                 VARCHAR              null,
   origin_path          VARCHAR              null,
   origin_name          VARCHAR              null,
   addi                 JSONB                null,
   status               VARCHAR              null,
   constraint PK_T_FILE primary key (id)
);

comment on table t_file is
'文件描述表';

comment on column t_file.id is
'编号';

comment on column t_file.file_oid is
'文件数据库对象编号';

comment on column t_file.file_name is
'文件名';

comment on column t_file.path is
'服务端存储文件路径';

comment on column t_file.belongto_path is
'分类(以路径方式表述)';

comment on column t_file.digest is
'sha512 digest';

comment on column t_file.size is
'文件大小';

comment on column t_file.create_time is
'创建时间';

comment on column t_file.creator is
'上传者';

comment on column t_file.domain_id is
'数据隶属';

comment on column t_file.count is
'文件引用计数';

comment on column t_file.belongto is
'隶属的对象编号';

comment on column t_file.limn is
'文件作用描述';

comment on column t_file.origin_path is
'用户上传文件路径';

comment on column t_file.origin_name is
'用户上传文件名';

comment on column t_file.addi is
'附加信息';

comment on column t_file.status is
'0:有效, 2: 丢失';

ALTER SEQUENCE t_file_id_seq RESTART WITH 20000;

/*==============================================================*/
/* Table: t_group                                               */
/*==============================================================*/
create table if not exists  t_group (
   id                   SERIAL not null,
   name                 VARCHAR              not null,
   bulletin             VARCHAR              null,
   admin                JSONB                null,
   owner                INT8                 null,
   naming_by_admin      BOOL                 null,
   invitation_need      BOOL                 null,
   realm                VARCHAR              not null,
   creator              INT8                 not null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   updated_by           INT8                 null,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   domain_id            INT8                 null,
   addi                 JSONB                null,
   remark               VARCHAR              null,
   status               VARCHAR              null default '01',
   constraint PK_T_GROUP primary key (id)
);

comment on table t_group is
'聊天群，设计参考微信群';

comment on column t_group.id is
'编号';

comment on column t_group.name is
'名称';

comment on column t_group.bulletin is
'群公告';

comment on column t_group.admin is
'群管理员列表，[user_id1,user_id2]';

comment on column t_group.owner is
'群主';

comment on column t_group.naming_by_admin is
'仅群管理员可改群名称';

comment on column t_group.invitation_need is
'邀请进群';

comment on column t_group.realm is
'群组类型, im: 聊天, class: 班级, auth: 权限';

comment on column t_group.creator is
'本数据创建者';

comment on column t_group.create_time is
'生成时间';

comment on column t_group.updated_by is
'更新者';

comment on column t_group.update_time is
'帐号信息更新时间';

comment on column t_group.domain_id is
'数据隶属';

comment on column t_group.addi is
'附加信息';

comment on column t_group.remark is
'备注';

comment on column t_group.status is
'状态，00：草稿，01：有效，02：作废';

/*==============================================================*/
/* Table: t_import_data                                         */
/*==============================================================*/
create table if not exists  t_import_data (
   id                   SERIAL not null,
   name                 VARCHAR              null,
   category             VARCHAR              not null,
   key                  VARCHAR              not null,
   entity_id            VARCHAR              null,
   struct               JSONB                not null,
   base                 JSONB                null,
   data                 JSONB                not null,
   file                 JSONB                null,
   file_digest          VARCHAR              not null,
   domain_id            INT8                 null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   creator              INT8                 null,
   updated_by           INT8                 null,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   addi                 JSONB                null,
   remark               VARCHAR              null,
   status               VARCHAR              null,
   constraint PK_T_IMPORT_DATA primary key (id)
);

comment on table t_import_data is
'excel导入表';

comment on column t_import_data.id is
'参数编号';

comment on column t_import_data.name is
'导入数据名称';

comment on column t_import_data.category is
'导入数据类型';

comment on column t_import_data.key is
'数据唯一标识, 由struct中的key简单连接组成';

comment on column t_import_data.entity_id is
'表示本条数据的逻辑标识，如，保单号，身份证号';

comment on column t_import_data.struct is
'excel数据结构';

comment on column t_import_data.base is
'表中的非重复信息';

comment on column t_import_data.data is
'数据';

comment on column t_import_data.file is
'导入文件信息';

comment on column t_import_data.file_digest is
'file_digest';

comment on column t_import_data.domain_id is
'数据隶属';

comment on column t_import_data.create_time is
'生成时间';

comment on column t_import_data.creator is
'本数据创建者';

comment on column t_import_data.updated_by is
'更新者';

comment on column t_import_data.update_time is
'帐号信息更新时间';

comment on column t_import_data.addi is
'附加信息';

comment on column t_import_data.remark is
'备注';

comment on column t_import_data.status is
'状态，00：草稿，01：有效，02：作废';

/*==============================================================*/
/* Index: idx_impdata_key                                       */
/*==============================================================*/
create unique index if not exists  idx_impdata_key on t_import_data (
category,
key
);

/*==============================================================*/
/* Index: idx_impdata_digest                                    */
/*==============================================================*/
create  index if not exists  idx_impdata_digest on t_import_data (
file_digest
);

/*==============================================================*/
/* Index: idx_impdata_entity_id                                 */
/*==============================================================*/
create  index if not exists  idx_impdata_entity_id on t_import_data (
entity_id
);

/*==============================================================*/
/* Table: t_insurance_policy                                    */
/*==============================================================*/
create table if not exists  t_insurance_policy (
   id                   SERIAL not null,
   sn                   VARCHAR              null,
   sn_creator           INT8                 null,
   name                 VARCHAR              not null,
   order_id             INT8                 not null,
   policy               VARCHAR              not null,
   start                INT8                 not null,
   cease                INT8                 not null,
   year                 INT2                 null,
   duration             INT8                 null,
   premium              FLOAT8               not null,
   third_party_premium  FLOAT8               null,
   third_party_account  VARCHAR              null,
   pay_time             INT8                 null,
   pay_channel          VARCHAR              null,
   pay_type             VARCHAR              null,
   unit_price           FLOAT8               null,
   org_id               INT8                 null,
   org_manager_id       INT8                 null,
   policyholder_type    VARCHAR              null,
   policyholder         JSONB                null,
   policyholder_id      INT8                 null,
   insurance_type       VARCHAR              null,
   insurance_type_id    INT8                 null,
   policy_scheme        JSONB                null,
   activity_name        VARCHAR              null,
   activity_category    VARCHAR              null,
   activity_desc        VARCHAR              null,
   activity_location    VARCHAR              null,
   activity_date_set    VARCHAR              null,
   insured_count        INT2                 null,
   compulsory_student_num INT8                 null,
   non_compulsory_student_num INT8                 null,
   contact              JSONB                null,
   fee_scheme           JSONB                null,
   car_service_target   VARCHAR              null,
   same                 BOOL                 null,
   relation             VARCHAR              null,
   insured              JSONB                null,
   insured_id           INT8                 null,
   have_insured_list    BOOL                 null,
   insured_group_by_day BOOL                 null,
   insured_type         VARCHAR              null,
   insured_list         JSONB                null,
   indate               INT8                 null,
   jurisdiction         VARCHAR              null,
   dispute_handling     VARCHAR              null,
   prev_policy_no       VARCHAR              null,
   insure_base          VARCHAR              null,
   blanket_insure_code  VARCHAR              null,
   custom_type          VARCHAR              null,
   train_projects       VARCHAR              null,
   business_locations   JSONB                null,
   arbitral_agency      VARCHAR              null,
   pool_num             INT2                 null,
   open_pool_num        INT2                 null,
   heated_pool_num      INT2                 null,
   training_pool_num    INT2                 null,
   inner_area           FLOAT8               null,
   outer_area           FLOAT8               null,
   pool_name            VARCHAR              null,
   have_dinner_num      BOOL                 null,
   dinner_num           INT4                 null,
   canteen_num          INT4                 null,
   shop_num             INT4                 null,
   have_rides           BOOL                 null,
   have_explosive       BOOL                 null,
   area                 INT4                 null,
   traffic_num          INT4                 null,
   temperature_type     VARCHAR              null,
   is_indoor            VARCHAR              null,
   extra                JSONB                null,
   bank_account         JSONB                null,
   pay_contact          VARCHAR              null,
   have_sudden_death    BOOL                 null,
   sudden_death_terms   VARCHAR              null,
   spec_agreement       VARCHAR              null,
   reminders_num        INT2                 null,
   is_entry_policy      BOOL                 null,
   is_admin_pay         BOOL                 null,
   policy_enroll_time   INT8                 null,
   zero_pay_status      VARCHAR              null,
   external_status      VARCHAR              null,
   cancel_desc          VARCHAR              null,
   favorite             BOOL                 null,
   creator              INT8                 null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   updated_by           INT8                 null,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   domain_id            INT8                 null,
   addi                 JSONB                null,
   remark               VARCHAR              null,
   status               VARCHAR              null,
   constraint PK_T_INSURANCE_POLICY primary key (id)
);

comment on table t_insurance_policy is
'保险单';

comment on column t_insurance_policy.id is
'编号';

comment on column t_insurance_policy.sn is
'保单号';

comment on column t_insurance_policy.sn_creator is
'保单号上传者ID';

comment on column t_insurance_policy.name is
'保险种类';

comment on column t_insurance_policy.order_id is
'订单编号';

comment on column t_insurance_policy.policy is
'保险合同条款';

comment on column t_insurance_policy.start is
'起保时间';

comment on column t_insurance_policy.cease is
'终保时间';

comment on column t_insurance_policy.year is
'保单年份';

comment on column t_insurance_policy.duration is
'保障期限';

comment on column t_insurance_policy.premium is
'保费金额';

comment on column t_insurance_policy.third_party_premium is
'第三方保费金额';

comment on column t_insurance_policy.third_party_account is
'自动录单账号';

comment on column t_insurance_policy.pay_time is
'支付时间';

comment on column t_insurance_policy.pay_channel is
'校快保，泰合，近邻，人保，太平洋保险';

comment on column t_insurance_policy.pay_type is
'支付方式: 对公转账/在线支付/线下支付';

comment on column t_insurance_policy.unit_price is
'单价';

comment on column t_insurance_policy.org_id is
'关联机构编号';

comment on column t_insurance_policy.org_manager_id is
'关联机构管理人';

comment on column t_insurance_policy.policyholder_type is
'投保人类型';

comment on column t_insurance_policy.policyholder is
'投保人';

comment on column t_insurance_policy.policyholder_id is
'投保人编码';

comment on column t_insurance_policy.insurance_type is
'保险类型: 学生意外伤害险，活动/比赛险(旅游险),食品卫生责任险，教工责任险,校方责任险,实习生责任险,校车责任险,游泳池责任险';

comment on column t_insurance_policy.insurance_type_id is
'保险产品编码';

comment on column t_insurance_policy.policy_scheme is
'保险方案';

comment on column t_insurance_policy.activity_name is
'活动名称';

comment on column t_insurance_policy.activity_category is
'活动类型';

comment on column t_insurance_policy.activity_desc is
'活动描述';

comment on column t_insurance_policy.activity_location is
'活动地点';

comment on column t_insurance_policy.activity_date_set is
'具体活动日期，英文逗号隔开';

comment on column t_insurance_policy.insured_count is
'总数量/保障人数/车辆数';

comment on column t_insurance_policy.compulsory_student_num is
'义务教育学生人数（校方）';

comment on column t_insurance_policy.non_compulsory_student_num is
'非义务教育人数（校方）';

comment on column t_insurance_policy.contact is
'联系人';

comment on column t_insurance_policy.fee_scheme is
'计费标准/单价';

comment on column t_insurance_policy.car_service_target is
'校车服务对象';

comment on column t_insurance_policy.same is
'投保人与被保险人是同一人';

comment on column t_insurance_policy.relation is
'投保人与被保险人关系';

comment on column t_insurance_policy.insured is
'被保险人';

comment on column t_insurance_policy.insured_id is
'被保险人编号';

comment on column t_insurance_policy.have_insured_list is
'有被保险对象清单';

comment on column t_insurance_policy.insured_group_by_day is
'被保险对象按日期分组';

comment on column t_insurance_policy.insured_type is
'被保险人类型: 学生，非学生';

comment on column t_insurance_policy.insured_list is
'被保险对象清单';

comment on column t_insurance_policy.indate is
'有效期(天)';

comment on column t_insurance_policy.jurisdiction is
'司法管辖权';

comment on column t_insurance_policy.dispute_handling is
'争议处理';

comment on column t_insurance_policy.prev_policy_no is
'续保保单号';

comment on column t_insurance_policy.insure_base is
'承保基础';

comment on column t_insurance_policy.blanket_insure_code is
'统保代码';

comment on column t_insurance_policy.custom_type is
'场地使用性质:internal, open, both';

comment on column t_insurance_policy.train_projects is
'训练项目';

comment on column t_insurance_policy.business_locations is
'承保地址/区域范围/游泳池场地地址';

comment on column t_insurance_policy.arbitral_agency is
'仲裁机构';

comment on column t_insurance_policy.pool_num is
'游泳池个数';

comment on column t_insurance_policy.open_pool_num is
'对外开放游泳池数量';

comment on column t_insurance_policy.heated_pool_num is
'恒温游泳池数量';

comment on column t_insurance_policy.training_pool_num is
'培训游泳池数量';

comment on column t_insurance_policy.inner_area is
'室内面积';

comment on column t_insurance_policy.outer_area is
'室外面积';

comment on column t_insurance_policy.pool_name is
'游泳池名称(英文逗号分隔) ';

comment on column t_insurance_policy.have_dinner_num is
'是否开启就餐人数';

comment on column t_insurance_policy.dinner_num is
'用餐人数';

comment on column t_insurance_policy.canteen_num is
'食堂个数';

comment on column t_insurance_policy.shop_num is
'商店个数';

comment on column t_insurance_policy.have_rides is
'营业场所是否有游泳池外游乐设施、机械性游乐设施等';

comment on column t_insurance_policy.have_explosive is
'营业场所是否有制造、销售、储存易燃易爆危险品';

comment on column t_insurance_policy.area is
'营业场所总面积（平方米）';

comment on column t_insurance_policy.traffic_num is
'每日客流量（人）';

comment on column t_insurance_policy.temperature_type is
'泳池性质:恒温、常温';

comment on column t_insurance_policy.is_indoor is
'泳池特性:室内、室外';

comment on column t_insurance_policy.extra is
'附加信息:
附加条款
企业经营描述
相关保险情况
保险公司提示
保险销售事项确认书
保险公司信息：经办人/工号、代理点代码、展业方式
产险销售人员：姓名、职业证号';

comment on column t_insurance_policy.bank_account is
'对公帐号信息：户名、所在银行、账号';

comment on column t_insurance_policy.pay_contact is
'线下支付联系人：微信二维码，base64';

comment on column t_insurance_policy.have_sudden_death is
'是否开启猝死责任险';

comment on column t_insurance_policy.sudden_death_terms is
'猝死条款内容：附加猝死保险责任每人限额5万元，累计限额5万元。附加猝死责任保险条款（经法院判决、仲裁机构裁决或根据县级以上政府及县级以上政府有关部门的行政决定书或者调解证明等材料，需由被保险人承担的经济赔偿责任，由保险人负责赔偿）';

comment on column t_insurance_policy.spec_agreement is
'特别约定';

comment on column t_insurance_policy.reminders_num is
'催款次数';

comment on column t_insurance_policy.is_entry_policy is
'保单是否已录入承保公司系统';

comment on column t_insurance_policy.is_admin_pay is
'管理员是否支付';

comment on column t_insurance_policy.policy_enroll_time is
'录单时间';

comment on column t_insurance_policy.zero_pay_status is
'0元实缴状态, 00: 未撤单, 02: 待实缴 04: 已实缴 06: 原保单已支付，不实缴';

comment on column t_insurance_policy.external_status is
'保单外部状态, 00: 待撤单, 02:撤单成功, 04:撤单失败';

comment on column t_insurance_policy.cancel_desc is
'撤单类型,04 重新录单 08撤销 20 拒保 24 退保';

comment on column t_insurance_policy.favorite is
'收藏';

comment on column t_insurance_policy.creator is
'创建者用户ID';

comment on column t_insurance_policy.create_time is
'创建时间';

comment on column t_insurance_policy.updated_by is
'更新者';

comment on column t_insurance_policy.update_time is
'更新时间';

comment on column t_insurance_policy.domain_id is
'数据属主';

comment on column t_insurance_policy.addi is
'附加数据';

comment on column t_insurance_policy.remark is
'备注';

comment on column t_insurance_policy.status is
'一期，0：受理中，2：在保，4：过保, 6: 作废。二期，00: 正常, 04: 重新录单, 08: 撤消, 12: 续保, 16: 已重新录单, 20: 退保, 24: 拒保';

ALTER SEQUENCE t_insurance_policy_id_seq RESTART WITH 20000;



/*==============================================================*/
/* Index: idx_insure_policy_SN                                  */
/*==============================================================*/
create  index if not exists  idx_insure_policy_SN on t_insurance_policy (
sn
);

/*==============================================================*/
/* Index: idx_insure_policy_order_id                            */
/*==============================================================*/
create  index if not exists  idx_insure_policy_order_id on t_insurance_policy (
order_id
);

/*==============================================================*/
/* Index: idx_insure_policy_status                              */
/*==============================================================*/
create  index if not exists  idx_insure_policy_status on t_insurance_policy (
status
);

/*==============================================================*/
/* Table: t_insurance_types                                     */
/*==============================================================*/
create table if not exists  t_insurance_types (
   id                   SERIAL not null,
   ref_id               INT8                 null,
   name                 VARCHAR              not null,
   alias                VARCHAR              null,
   data_type            VARCHAR              null,
   parent_id            INT8                 null,
   age_limit            JSONB                null,
   rule_batch           VARCHAR              null,
   org_id               INT8                 null,
   pay_type             VARCHAR              null,
   pay_channel          VARCHAR              null,
   pay_name             VARCHAR              null,
   bank_account         VARCHAR              null,
   bank_account_name    VARCHAR              null,
   bank_name            VARCHAR              null,
   bank_id              VARCHAR              null,
   floor_price          FLOAT8               null,
   unit_price           FLOAT8               null,
   price                FLOAT8               null,
   price_config         JSONB                null,
   define_level         INT2                 not null,
   layout_order         INT2                 null,
   layout_level         INT2                 not null,
   list_tpl             VARCHAR              null,
   files                JSONB                null,
   resource             JSONB                null,
   pic                  VARCHAR              null,
   sudden_death_description JSONB                null,
   description          VARCHAR              null,
   auto_fill            VARCHAR              null,
   enable_import_list   BOOL                 null,
   have_dinner_num      BOOL                 null,
   invoice_title_update_times INT2                 null,
   receipt_account      JSONB                null,
   transfer_auth_files  JSONB                null,
   contact              JSONB                null,
   contact_qr_code      VARCHAR              null,
   other_files          JSONB                null,
   insurer              VARCHAR              null,
   underwriter          JSONB                null,
   remind_days          INT2                 null,
   mail                 JSONB                null,
   order_repeat_limit   INT2                 null,
   group_by_max_day     INT2                 null,
   web_description      VARCHAR              null,
   mobile_description   VARCHAR              null,
   auto_fill_param      JSONB                null,
   "interval"           INT8                 null,
   max_insure_in_year   INT2                 null,
   insured_in_month     INT2                 null,
   insured_start_time   INT8                 null,
   insured_end_time     INT8                 null,
   allow_start          INT8                 null,
   allow_end            INT8                 null,
   indate_start         INT8                 null,
   indate_end           INT8                 null,
   creator              VARCHAR              null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   updated_by           INT8                 null,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   addi                 JSONB                null,
   domain_id            INT8                 null,
   remark               VARCHAR              null,
   status               VARCHAR              null,
   constraint PK_T_INSURANCE_TYPES primary key (id)
);

comment on table t_insurance_types is
'保险类型表';

comment on column t_insurance_types.id is
'保险产品id';

comment on column t_insurance_types.ref_id is
'机构引用的保险方案编码，即本表 org_id=0 and data_type="4" and parent_id=各险种ID的数据';

comment on column t_insurance_types.name is
'保险产品名称';

comment on column t_insurance_types.alias is
'别名';

comment on column t_insurance_types.data_type is
'0: 保险产品分类, 2: 保险产品定义, 4: 投保规则, 6: 保险方案, 8: 默认投保规则/方案';

comment on column t_insurance_types.parent_id is
'隶属保险产品分类，0：表示无上级分类';

comment on column t_insurance_types.age_limit is
'年龄限制';

comment on column t_insurance_types.rule_batch is
'规则批次';

comment on column t_insurance_types.org_id is
'投保规则、方案所属的机构编码';

comment on column t_insurance_types.pay_type is
'支付方式: 对公转账/在线支付/线下支付';

comment on column t_insurance_types.pay_channel is
'校快保，泰合，近邻，人保，太平洋保险';

comment on column t_insurance_types.pay_name is
'支付项显示的名称';

comment on column t_insurance_types.bank_account is
'收款银行账号';

comment on column t_insurance_types.bank_account_name is
'收款户名';

comment on column t_insurance_types.bank_name is
'开户行名称';

comment on column t_insurance_types.bank_id is
'开户行行号';

comment on column t_insurance_types.floor_price is
'首页显示的最低价';

comment on column t_insurance_types.unit_price is
'单价';

comment on column t_insurance_types.price is
'价格（分）';

comment on column t_insurance_types.price_config is
'价格方案';

comment on column t_insurance_types.define_level is
'保险产品实际层次';

comment on column t_insurance_types.layout_order is
'保险产品显示顺序';

comment on column t_insurance_types.layout_level is
'保险产品显示层次';

comment on column t_insurance_types.list_tpl is
'清单模板';

comment on column t_insurance_types.files is
'清单模板';

comment on column t_insurance_types.resource is
'资源';

comment on column t_insurance_types.pic is
'关联图片';

comment on column t_insurance_types.sudden_death_description is
'猝死责任险描述';

comment on column t_insurance_types.description is
'首页描述';

comment on column t_insurance_types.auto_fill is
'第三方录单(比赛险-人保录单), 0: 不自动录单，2：自动录单';

comment on column t_insurance_types.enable_import_list is
'允许录入清单';

comment on column t_insurance_types.have_dinner_num is
'是否开启就餐人数';

comment on column t_insurance_types.invoice_title_update_times is
'发票抬头修改次数设置';

comment on column t_insurance_types.receipt_account is
'对公账号设置,例:{"户名":"广州校快保科技有限公司 ",
"开户行":"中国银行",
"账号":"45641857894861548979"}';

comment on column t_insurance_types.transfer_auth_files is
'转账授权说明文件';

comment on column t_insurance_types.contact is
'协议价短信联系人,例:{"联系人":"张鸣","联系电话":18311706633}';

comment on column t_insurance_types.contact_qr_code is
'缴费联系人设置,存放二维码';

comment on column t_insurance_types.other_files is
'其它相关文件';

comment on column t_insurance_types.insurer is
'承保公司,用于方案规则';

comment on column t_insurance_types.underwriter is
'承保公司';

comment on column t_insurance_types.remind_days is
'自动催款天数';

comment on column t_insurance_types.mail is
'邮寄地址设置,例:{"收件人":"张鸣",
"联系电话":18311706633,
"邮寄地址":"广东省广州市番禺区大学城外环西路303号校快保科技有限公司"}';

comment on column t_insurance_types.order_repeat_limit is
'最大订单份数';

comment on column t_insurance_types.group_by_max_day is
'允许最多按天分组数';

comment on column t_insurance_types.web_description is
'PC页面描述';

comment on column t_insurance_types.mobile_description is
'移动端页面描述';

comment on column t_insurance_types.auto_fill_param is
'存放各个险的特定参数';

comment on column t_insurance_types."interval" is
'间隔时间';

comment on column t_insurance_types.max_insure_in_year is
'最长投保年限（年）';

comment on column t_insurance_types.insured_in_month is
'保障时长（月）';

comment on column t_insurance_types.insured_start_time is
'起保日期';

comment on column t_insurance_types.insured_end_time is
'止保日期';

comment on column t_insurance_types.allow_start is
'投保开始日期';

comment on column t_insurance_types.allow_end is
'投保结束日期';

comment on column t_insurance_types.indate_start is
'规则起效日期';

comment on column t_insurance_types.indate_end is
'规则失效日期';

comment on column t_insurance_types.creator is
'创建者';

comment on column t_insurance_types.create_time is
'创建时间';

comment on column t_insurance_types.updated_by is
'更新者';

comment on column t_insurance_types.update_time is
'更新时间';

comment on column t_insurance_types.addi is
'备用字段';

comment on column t_insurance_types.domain_id is
'数据属主';

comment on column t_insurance_types.remark is
'备注';

comment on column t_insurance_types.status is
'状态, 0: 正常，2:等待推出, 4：禁用，6：作废';


ALTER SEQUENCE t_insurance_types_id_seq RESTART WITH 20000;

------------ 比赛活动保险
INSERT INTO t_insurance_types (id, data_type, name, alias, parent_id, define_level, layout_order, layout_level,group_by_max_day,
 web_description, mobile_description, pic, floor_price, auto_fill, invoice_title_update_times, receipt_account, 
 contact, underwriter, auto_fill_param, remind_days, order_repeat_limit, age_limit, status) VALUES 
(10000, '0', '比赛/体考/活动保险(短期)', '2020年招生体育考试', 0, 2, 0, 0, 2,'完全符合主办方关于保险的要求，专业提供体育保险服务十余年。','符合主办方要求',
'/api/xkbPic?q=10000.jpg', 500, '人保录单', 2, 
'[{"Bank": "中国工商银行广州市第一支行", "Account": "3602000109001051277", "AccountName": "中国人民财产保险股份有限公司广州市分公司","BankNum":"102581000013"}]', 
'[{"Phone": "13189503871", "ContactName": "shadow"}]', 
 '[{"Account": "GD02000601", "Company": "人保", "EndTime": 1641394245000, "Password": "f32eff772b438a50c2cadb58bc06ce35", "StartTime": 1576108800000}]',
 '[{"Phone": "13925009308","Reviewer": "陈静茹"}]', 3, 3,
 '{"MaleMax":60,"MaleMin":2,"FemaleMax":55,"FemaleMin":2}',
  '0');

INSERT INTO t_insurance_types (id, data_type, name, parent_id, define_level, layout_order, layout_level, pic, order_repeat_limit,remark, status) VALUES 
(10002, '2', '体育比赛', 10000, 0, 2, 8, '/api/xkbPic?q=10002.svg',3, '比赛类', '0'),
(10004, '2', '科技比赛', 10000, 0, 6, 8, '/api/xkbPic?q=10004.svg', 3, '比赛类', '0'),
(10006, '2', '军训/国防教育', 10000, 0, 8, 8, '/api/xkbPic?q=10006.svg', 3, '活动类', '0'),
(10008, '2', '综合实践', 10000, 0, 8, 10, '/api/xkbPic?q=10008.svg', 3, '活动类', '0'),
(10010, '2', '体质健康测试', 10000, 0, 12, 8, '/api/xkbPic?q=10010.svg', 3, '活动类', '0'),
(10012, '2', '各类活动', 10000, 0, 14, 8, '/api/xkbPic?q=10012.svg',3, '活动类', '0'),
(10014, '2', '体育考试', 10000, 0, 4, 8, '/api/xkbPic?q=10014.svg',3, '比赛类', '0');

------------校园方责任保险




insert into t_insurance_types(parent_id,id,name,alias,pay_type,pay_channel,insurer,data_type,age_limit,pay_name,floor_price,unit_price,price,define_level,layout_order,layout_level,list_tpl,pic,have_dinner_num,receipt_account,contact,contact_qr_code,underwriter,mail,web_description,mobile_description,max_insure_in_year,insured_in_month,allow_start,allow_end,indate_start,indate_end,creator,addi,domain_id,remark,status) values
(0,10020,'校(园)方系列责任保险',null,null,null,null,'2','{"MaleMax":60,"MaleMin":2,"FemaleMax":55,"FemaleMin":2}',null,7,null,null,0,20,0,null,'/api/xkbPic?q=10020.jpg',null,null,null,null,null,null,'校责、教工、实习生、食品安全、校车等校园责任保险','多种校园方责任保险',null,null,null,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),

(10020,10022,'校(园)方责任保险',null,null,null,null,'0','{"MaleMax":60,"MaleMin":2,"FemaleMax":55,"FemaleMin":2}',null,7,null,null,2,22,4,'清单模板/校责险清单模板.xlsx','/api/xkbPic?q=10022.svg',FALSE,null,null,'/api/xkbPic?q=default_contact_qr_code.png',null,'[{"Phone": "13925001114","Address": "广州市海珠区宝岗大道北155号PICC二楼","Receiver": "周兰芬"}]',null,null,null,null,null,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),
(10020,10024,'教职员工校(园)方责任保险',null,null,null,null,'0','{"MaleMax":60,"MaleMin":2,"FemaleMax":55,"FemaleMin":2}',null,7,null,null,2,24,4,'清单模板/教职工清单模板.xlsx','/api/xkbPic?q=10024.svg',null,null,null,'/api/xkbPic?q=default_contact_qr_code.png',null,'[{"Phone": "13925001114","Address": "广州市海珠区宝岗大道北155号PICC二楼","Receiver": "周兰芬"}]',null,null,null,null,null,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),
(10020,10026,'职业院校学生实习责任保险',null,null,null,null,'0','{"MaleMax":60,"MaleMin":2,"FemaleMax":55,"FemaleMin":2}',null,7,null,null,2,28,4,'清单模板/实习生清单模板.xlsx','/api/xkbPic?q=10026.svg',null,null,'[{"Phone": "13189503871", "ContactName": "shadow"}]' ,'/api/xkbPic?q=default_contact_qr_code.png',null,'[{"Phone": "13925001114","Address": "广州市海珠区宝岗大道北155号PICC二楼","Receiver": "周兰芬"}]',null,null,null,null,null,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),
(10020,10028,'校车承运人责任保险',null,null,null,null,'0','{"MaleMax":60,"MaleMin":2,"FemaleMax":55,"FemaleMin":2}',null,7,null,null,2,30,4,'清单模板/校车清单模板.xlsx','/api/xkbPic?q=10028.svg',null,null,null,'/api/xkbPic?q=default_contact_qr_code.png',null,'[{"Phone": "13925001114","Address": "广州市海珠区宝岗大道北155号PICC二楼","Receiver": "周兰芬"}]',null,null,null,null,null,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),
(10020,10030,'餐饮场所责任保险',null,null,null,null,'0','{"MaleMax":60,"MaleMin":2,"FemaleMax":55,"FemaleMin":2}',null,7,null,null,2,26,4,null,'/api/xkbPic?q=10030.svg',FALSE,null,null,'/api/xkbPic?q=default_contact_qr_code.png',null,'[{"Phone": "13925001114","Address": "广州市海珠区宝岗大道北155号PICC二楼","Receiver": "周兰芬"}]',null,null,null,null,null,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),

(0,10040,'学生意外伤害险(期限一年起)',null,null,null,null,'0','{"MaleMax":60,"MaleMin":2,"FemaleMax":55,"FemaleMin":2}',null,10000,null,null,2,18,0,null,'/api/xkbPic?q=10040.jpg',null,null,null,null,null,null,'团体投保，保费低，保障高','保费低保障高',null,null,null,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),
(10040,10042,'幼儿园',null,null,null,null,'2',null,null,null,null,null,0,42,8,null,'/api/xkbPic?q=10042.png',null,null,null,null,null,null,null,null,null,null,null,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),
(10040,10044,'小学',null,null,null,null,'2',null,null,null,null,null,0,44,8,null,'/api/xkbPic?q=10044.png',null,null,null,null,null,null,null,null,null,null,null,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),
(10040,10046,'初中',null,null,null,null,'2',null,null,null,null,null,0,46,8,null,'/api/xkbPic?q=10046.png',null,null,null,null,null,null,null,null,null,null,null,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),
(10040,10048,'高中',null,null,null,null,'2',null,null,null,null,null,0,48,8,null,'/api/xkbPic?q=10048.png',null,null,null,null,null,null,null,null,null,null,null,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),
(10040,10050,'九年一贯制',null,null,null,null,'2',null,null,null,null,null,0,50,8,null, '/api/xkbPic?q=10050.png',null,null,null,null,null,null,null,null,null,null,null,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),
(10040,10052,'高职',null,null,null,null,'2',null,null,null,null,null,0,52,8,null,'/api/xkbPic?q=10052.png',null,null,null,null,null,null,null,null,null,null,null,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),
(10040,10054,'完中',null,null,null,null,'2',null,null,null,null,null,0,54,8,null,'/api/xkbPic?q=10054.png',null,null,null,null,null,null,null,null,null,null,null,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),
(10040,10056,'大学',null,null,null,null,'2',null,null,null,null,null,0,56,8,null,'/api/xkbPic?q=10056.png',null,null,null,null,null,null,null,null,null,null,null,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),
(10040,12002,'学生意外伤害险-近邻',null,'在线支付','近邻',null,'2',null,null,1,null,null,0,3,20,null,null,null,null,null,null,null,null,null,null,null,null,null,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),
(10040,12004,'学生意外伤害险-校快保',null,'在线支付','校快保',null,'2',null,null,1,null,null,0,2,20,null,null,null,null,null,null,null,null,null,null,null,null,null,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),
(10040,12006,'学生意外伤害险-泰合',null,'在线支付','泰合',null,'2',null,null,100,null,null,0,1,20,null,null,null,null,null,null,null,null,null,null,null,null,null,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),
(10040,12040,'学生意外伤害险-近邻','散单-近邻','在线支付','近邻','中国人民财产保险股份有限公司','4',null,'近邻',null,1,1,0,null,0,null,null,null,null,null,null,null,null,null,null,null,12,null,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),
(10040,12042,'学生意外伤害险-校快保','散单-校快保','在线支付','校快保','中国人民财产保险股份有限公司','4',null,'校快保',null,1,1,0,null,0,null,null,null,null,null,null,null,null,null,null,null,12,null,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),
(10040,12044,'学生意外伤害险-泰合','散单-泰合','在线支付','泰合','中国人民财产保险股份有限公司','4',null,'泰合',null,10000,10000,0,null,0,null,null,null,null,null,null,null,null,null,null,null,12,null,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),
(10040,12050,'学生意外伤害险-幼儿园','团单-幼儿园','在线支付','近邻','中国人民财产保险股份有限公司','4',null,'近邻',null,1,null,0,null,0,null,null,null,null,null,null,null,null,null,null,4,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 month'))*1000,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),
(10040,12052,'学生意外伤害险-小学','团单-小学','在线支付','近邻','中国人民财产保险股份有限公司','4',null,'近邻',null,1,null,0,null,0,null,null,null,null,null,null,null,null,null,null,6,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 month'))*1001,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),
(10040,12054,'学生意外伤害险-初中','团单-初中','在线支付','近邻','中国人民财产保险股份有限公司','4',null,'近邻',null,1,null,0,null,0,null,null,null,null,null,null,null,null,null,null,3,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 month'))*1002,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),
(10040,12058,'学生意外伤害险-高中','团单-高中','在线支付','近邻','中国人民财产保险股份有限公司','4',null,'近邻',null,1,null,0,null,0,null,null,null,null,null,null,null,null,null,null,3,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 month'))*1003,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),
(10040,12060,'学生意外伤害险-九年一贯制','团单-九年一贯制','在线支付','近邻','中国人民财产保险股份有限公司','4',null,'近邻',null,1,null,0,null,0,null,null,null,null,null,null,null,null,null,null,9,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 month'))*1004,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),
(10040,12062,'学生意外伤害险-高职','团单-高职','在线支付','近邻','中国人民财产保险股份有限公司','4',null,'近邻',null,1,null,0,null,0,null,null,null,null,null,null,null,null,null,null,3,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 month'))*1005,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),
(10040,12064,'学生意外伤害险-完中','团单-完中','在线支付','近邻','中国人民财产保险股份有限公司','4',null,'近邻',null,1,null,0,null,0,null,null,null,null,null,null,null,null,null,null,6,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 month'))*1006,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),
(10040,12066,'学生意外伤害险-大学','团单-大学','在线支付','近邻','中国人民财产保险股份有限公司','4',null,'近邻',null,1,null,0,null,0,null,null,null,null,null,null,null,null,null,null,4,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 month'))*1007,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0');


insert into t_insurance_types(parent_id,id,name,alias,pay_type,pay_channel,insurer,data_type,age_limit,pay_name,floor_price,unit_price,price,define_level,layout_order,layout_level,list_tpl,pic,have_dinner_num,receipt_account,contact,contact_qr_code,underwriter,mail,web_description,mobile_description,max_insure_in_year,insured_in_month,allow_start,allow_end,indate_start,indate_end,creator,addi,domain_id,remark,status) values
(10022,12070,'校(园)方责任保险','散单-校(园)方责任保险-对公转账','公对公转账','校快保','中国人民财产保险股份有限公司','6','{"MaleMax":60,"MaleMin":2,"FemaleMax":55,"FemaleMin":2}','对公转账',null,null,null,0,null,0,'清单模板/校责险清单模板.xlsx',null,FALSE,'[{"Bank": "中国工商银行广州市第一支行", "Account": "3602000109001051277", "AccountName": "中国人民财产保险股份有限公司广州市分公司","BankNum":"102581000013"}]',null,'/api/xkbPic?q=default_contact_qr_code.png',null,'[{"Phone": "13925001114","Address": "广州市海珠区宝岗大道北155号PICC二楼","Receiver": "周兰芬"}]',null,null,null,null,null,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),
(10024,12072,'教职员工校(园)方责任保险','散单-教职员工校(园)方责任保险-对公转账','公对公转账','校快保','中国人民财产保险股份有限公司','6','{"MaleMax":60,"MaleMin":2,"FemaleMax":55,"FemaleMin":2}','对公转账',null,null,null,0,null,0,'清单模板/教职工清单模板.xlsx',null,null,'[{"Bank": "中国工商银行广州市第一支行", "Account": "3602000109001051277", "AccountName": "中国人民财产保险股份有限公司广州市分公司","BankNum":"102581000013"}]',null,'/api/xkbPic?q=default_contact_qr_code.png',null,'[{"Phone": "13925001114","Address": "广州市海珠区宝岗大道北155号PICC二楼","Receiver": "周兰芬"}]',null,null,null,null,null,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),
(10026,12074,'职业院校学生实习责任保险','散单-职业院校学生实习责任保险-对公转账','公对公转账','校快保','中国人民财产保险股份有限公司','6','{"MaleMax":60,"MaleMin":2,"FemaleMax":55,"FemaleMin":2}','对公转账',null,null,null,0,null,0,'清单模板/实习生清单模板.xlsx',null,null,'[{"Bank": "中国工商银行广州市第一支行", "Account": "3602000109001051277", "AccountName": "中国人民财产保险股份有限公司广州市分公司","BankNum":"102581000013"}]','[{"Phone": "13189503871", "ContactName": "shadow"}]' ,'/api/xkbPic?q=default_contact_qr_code.png',null,'[{"Phone": "13925001114","Address": "广州市海珠区宝岗大道北155号PICC二楼","Receiver": "周兰芬"}]',null,null,null,null,null,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),
(10028,12076,'校车承运人责任保险','散单-校车承运人责任保险-对公转账','公对公转账','校快保','中国人民财产保险股份有限公司','6','{"MaleMax":60,"MaleMin":2,"FemaleMax":55,"FemaleMin":2}','对公转账',null,null,null,0,null,0,'清单模板/校车清单模板.xlsx',null,null,'[{"Bank": "中国工商银行广州市第一支行", "Account": "3602000109001051277", "AccountName": "中国人民财产保险股份有限公司广州市分公司","BankNum":"102581000013"}]',null,'/api/xkbPic?q=default_contact_qr_code.png',null,'[{"Phone": "13925001114","Address": "广州市海珠区宝岗大道北155号PICC二楼","Receiver": "周兰芬"}]',null,null,null,null,null,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),
(10030,12078,'餐饮场所责任保险','散单-餐饮场所责任保险-对公转账','公对公转账','校快保','中国人民财产保险股份有限公司','6','{"MaleMax":60,"MaleMin":2,"FemaleMax":55,"FemaleMin":2}','对公转账',null,null,null,0,null,0,null,null,FALSE,'[{"Bank": "中国工商银行广州市第一支行", "Account": "3602000109001051277", "AccountName": "中国人民财产保险股份有限公司广州市分公司","BankNum":"102581000013"}]',null,'/api/xkbPic?q=default_contact_qr_code.png',null,'[{"Phone": "13925001114","Address": "广州市海珠区宝岗大道北155号PICC二楼","Receiver": "周兰芬"}]',null,null,null,null,null,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),

(10022,12090,'校(园)方责任保险','散单-校(园)方责任保险-线下支付','线下支付','校快保','中国人民财产保险股份有限公司','6','{"MaleMax":60,"MaleMin":2,"FemaleMax":55,"FemaleMin":2}','线下支付',null,null,null,0,null,0,'清单模板/校责险清单模板.xlsx',null,FALSE,'[{"Bank": "中国工商银行广州市第一支行", "Account": "3602000109001051277", "AccountName": "中国人民财产保险股份有限公司广州市分公司","BankNum":"102581000013"}]',null,'/api/xkbPic?q=default_contact_qr_code.png',null,'[{"Phone": "13925001114","Address": "广州市海珠区宝岗大道北155号PICC二楼","Receiver": "周兰芬"}]',null,null,null,null,null,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),
(10024,12092,'教职员工校(园)方责任保险','散单-教职员工校(园)方责任保险-线下支付','线下支付','校快保','中国人民财产保险股份有限公司','6','{"MaleMax":60,"MaleMin":2,"FemaleMax":55,"FemaleMin":2}','线下支付',null,null,null,0,null,0,'清单模板/教职工清单模板.xlsx',null,null,'[{"Bank": "中国工商银行广州市第一支行", "Account": "3602000109001051277", "AccountName": "中国人民财产保险股份有限公司广州市分公司","BankNum":"102581000013"}]',null,'/api/xkbPic?q=default_contact_qr_code.png',null,'[{"Phone": "13925001114","Address": "广州市海珠区宝岗大道北155号PICC二楼","Receiver": "周兰芬"}]',null,null,null,null,null,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),
(10026,12094,'职业院校学生实习责任保险','散单-职业院校学生实习责任保险-线下支付','线下支付','校快保','中国人民财产保险股份有限公司','6','{"MaleMax":60,"MaleMin":2,"FemaleMax":55,"FemaleMin":2}','线下支付',null,null,null,0,null,0,'清单模板/实习生清单模板.xlsx',null,null,'[{"Bank": "中国工商银行广州市第一支行", "Account": "3602000109001051277", "AccountName": "中国人民财产保险股份有限公司广州市分公司","BankNum":"102581000013"}]','[{"Phone": "13189503871", "ContactName": "shadow"}]' ,'/api/xkbPic?q=default_contact_qr_code.png',null,'[{"Phone": "13925001114","Address": "广州市海珠区宝岗大道北155号PICC二楼","Receiver": "周兰芬"}]',null,null,null,null,null,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),
(10028,12096,'校车承运人责任保险','散单-校车承运人责任保险-线下支付','线下支付','校快保','中国人民财产保险股份有限公司','6','{"MaleMax":60,"MaleMin":2,"FemaleMax":55,"FemaleMin":2}','线下支付',null,null,null,0,null,0,'清单模板/校车清单模板.xlsx',null,null,'[{"Bank": "中国工商银行广州市第一支行", "Account": "3602000109001051277", "AccountName": "中国人民财产保险股份有限公司广州市分公司","BankNum":"102581000013"}]',null,'/api/xkbPic?q=default_contact_qr_code.png',null,'[{"Phone": "13925001114","Address": "广州市海珠区宝岗大道北155号PICC二楼","Receiver": "周兰芬"}]',null,null,null,null,null,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0'),
(10030,12098,'餐饮场所责任保险','散单-餐饮场所责任保险-线下支付','线下支付','校快保','中国人民财产保险股份有限公司','6','{"MaleMax":60,"MaleMin":2,"FemaleMax":55,"FemaleMin":2}','线下支付',null,null,null,0,null,0,null,null,FALSE,'[{"Bank": "中国工商银行广州市第一支行", "Account": "3602000109001051277", "AccountName": "中国人民财产保险股份有限公司广州市分公司","BankNum":"102581000013"}]',null,'/api/xkbPic?q=default_contact_qr_code.png',null,'[{"Phone": "13925001114","Address": "广州市海珠区宝岗大道北155号PICC二楼","Receiver": "周兰芬"}]',null,null,null,null,null,null,(extract('epoch' from current_timestamp)*1000)::bigint,extract('epoch' from (current_timestamp + interval '1 year'))*1000,1002,'{"init":true}',1000,null,'0');
insert into t_insurance_types(parent_id,id,name,ref_id,org_id,allow_start,allow_end,define_level,layout_level,status,data_type) values
(10022,12080,'校(园)方责任保险',12070,0,1,4102416000000,0,0,'0','8'),
(10024,12082,'教职员工校(园)方责任保险',12072,0,1,4102416000000,0,0,'0','8'),
(10026,12084,'职业院校学生实习责任保险',12074,0,1,4102416000000,0,0,'0','8'),
(10028,12086,'校车承运人责任保险',12076,0,1,4102416000000,0,0,'0','8'),
(10030,12088,'餐饮场所责任保险',12078,0,1,4102416000000,0,0,'0','8'),
(10022,12100,'校(园)方责任险',12090,0,1,4102416000000,0,0,'0','8'),
(10024,12102,'教职员工校(园)方责任险',12092,0,1,4102416000000,0,0,'0','8'),
(10026,12104,'职业院校学生实习责任险',12094,0,1,4102416000000,0,0,'0','8'),
(10028,12106,'校车承运人责任险',12096,0,1,4102416000000,0,0,'0','8'),
(10030,12108,'餐饮场所责任险',12098,0,1,4102416000000,0,0,'0','8');
update t_insurance_types set resource = '{
	"投保须知":	"",
	"保险条款":	"",
	"免责和退保声明":	"",
	"特别约定":	"",
	"time_set":{
		"投保须知":0,
		"保险条款":0,
		"免责和退保声明":0,
		"特别约定":0
	}
}' where id > 12000;

------太平洋责任保险
 INSERT INTO t_insurance_types (id, data_type, name, parent_id, define_level, layout_order, layout_level,web_description, mobile_description, pic, floor_price, contact_qr_code, status) VALUES 
 (10060, '0', '比赛/活动组织方责任保险', 0, 2, 60, 0, '保额高，保费低，大大减少组织方承担的风险。','有效转移组织方风险',
 '/api/xkbPic?q=10060.jpg', 200000, '/api/xkbPic?q=default_contact_qr_code.png', '0'),
 (10070, '0', '俱乐部/场地责任保险', 0, 2, 70, 0, '防范重大事故，保障俱乐部正常活动。','减少重大事故损失', 
 '/api/xkbPic?q=10070.jpg', 120000, '/api/xkbPic?q=default_contact_qr_code.png', '0'),
 (10080, '0', '游泳池责任保险', 0, 2, 80, 0, '风险高发，作用重大，每人保额50万元起。','事故高发作用重大',
 '/api/xkbPic?q=10080.jpg', 500000,'/api/xkbPic?q=default_contact_qr_code.png','0'),
 (10090, '0', '健康险', 0, 2, 90, 0,  '各大寿险公司，重大疾病等健康险精选方案。','精选寿险产品',
 '/api/xkbPic?q=10090.jpeg', 0,'/api/xkbPic?q=default_contact_qr_code.png', '2');

update t_insurance_types 
set receipt_account = '[{"Bank": "中国工商银行广州市第一支行", "Account": "3602000109001051277", "BankNum": "102581000013", "AccountName": "中国人民财产保险股份有限公司广州市分公司"}]',
contact = '[{"Phone": "13189503871", "ContactName": "shadow"}]',
underwriter = '[{"Account": "GD02000601", "Company": "人保", "EndTime": 1641394245000, "Password": "f32eff772b438a50c2cadb58bc06ce35", "StartTime": 1576108800000}]',
remind_days = 2,mail = '[{"Phone": "13925001114","Address": "广州市海珠区宝岗大道北155号PICC二楼","Receiver": "周兰芬"}]' where id in (10060,10070,10080);


update t_insurance_types set interval = 10 where id = 10060;
update t_insurance_types set sudden_death_description ='{
"false":"否：本保单不包含猝死责任。",
"true":"是：附加猝死保险责任每人限额5万元，累计限额5万元。附加猝死责任保险条款（经法院判决、仲裁机构裁决或根据县级以上政府及县级以上政府有关部门的行政决定书或者调解证明等材料，需由被保险人承担的经济赔偿责任，由保险人负责赔偿）"}'
where id in (10060,10070);



update t_insurance_types set resource = '{
    "自愿购买同意书":"",
    "保险介绍":"",
    "time_set":{
        "自愿购买同意书":0,
        "保险介绍":0
    }
}' where id = 10040;

update t_insurance_types set resource = '{
    "保险介绍":"",
    "time_set":{"保险介绍":0}
}' where id in (10000,10022,10024,10026,10028,10030,10060,10070,10080);


/*==============================================================*/
/* Index: idx_insure_type_refid_orgid                           */
/*==============================================================*/
create unique index if not exists  idx_insure_type_refid_orgid on t_insurance_types (
org_id,
ref_id
);

/*==============================================================*/
/* Index: idx_insure_type_channel                               */
/*==============================================================*/
create unique index if not exists  idx_insure_type_channel on t_insurance_types (
name,
pay_type,
pay_channel,
insurer
);

/*==============================================================*/
/* Table: t_insure_attach                                       */
/*==============================================================*/
create table if not exists  t_insure_attach (
   id                   SERIAL not null,
   t_u_id               INT8                 null,
   school_id            INT8                 not null,
   grade                VARCHAR              null,
   year                 INT2                 null,
   batch                VARCHAR              null,
   policy_no            VARCHAR              null,
   insure_policy_id     INT8                 null,
   others               JSONB                null,
   files                JSONB                null,
   creator              INT8                 null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   updated_by           INT8                 null,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   addi                 JSONB                null,
   domain_id            INT8                 null,
   remark               VARCHAR              null,
   status               VARCHAR              null,
   constraint PK_T_INSURE_ATTACH primary key (id)
);

comment on table t_insure_attach is
'保单附件';

comment on column t_insure_attach.id is
'编号';

comment on column t_insure_attach.t_u_id is
'用户内部编号';

comment on column t_insure_attach.school_id is
'学校编号';

comment on column t_insure_attach.grade is
'年级';

comment on column t_insure_attach.year is
'保单年份';

comment on column t_insure_attach.batch is
'批次';

comment on column t_insure_attach.policy_no is
'保单号';

comment on column t_insure_attach.insure_policy_id is
'系统保单编号';

comment on column t_insure_attach.others is
'其它';

comment on column t_insure_attach.files is
'保单附件';

comment on column t_insure_attach.creator is
'创建者用户ID';

comment on column t_insure_attach.create_time is
'创建时间';

comment on column t_insure_attach.updated_by is
'更新者';

comment on column t_insure_attach.update_time is
'修改时间';

comment on column t_insure_attach.addi is
'附加数据';

comment on column t_insure_attach.domain_id is
'数据属主';

comment on column t_insure_attach.remark is
'备注';

comment on column t_insure_attach.status is
'状态';

ALTER SEQUENCE t_insure_attach_id_seq RESTART WITH 20000;

/*==============================================================*/
/* Index: idx_insure_attach                                     */
/*==============================================================*/
create unique index if not exists  idx_insure_attach on t_insure_attach (
school_id,
grade,
batch,
year
);

/*==============================================================*/
/* Index: idx_insure_attach_p                                   */
/*==============================================================*/
create unique index if not exists  idx_insure_attach_p on t_insure_attach (
school_id,
insure_policy_id
);

/*==============================================================*/
/* Table: t_insured_detail                                      */
/*==============================================================*/
create table if not exists  t_insured_detail (
   id                   SERIAL not null,
   type                 VARCHAR              null,
   sub_type             VARCHAR              null,
   order_id             INT8                 null,
   policy_id            VARCHAR              null,
   name                 VARCHAR              null,
   id_card_no           VARCHAR              null,
   gender               VARCHAR              null,
   birthday             INT8                 null,
   role                 VARCHAR              null,
   org                  VARCHAR              null,
   class                VARCHAR              null,
   group_day            INT8                 null,
   license_plate_no     VARCHAR              null,
   brand                VARCHAR              null,
   driver_seat_number   INT2                 null,
   approved_passengers_num INT2                 null,
   seat_num             INT2                 null,
   road_grade           VARCHAR              null,
   driver_license       VARCHAR              null,
   driving_license      VARCHAR              null,
   action               VARCHAR              null,
   err_msg              VARCHAR              null,
   province             VARCHAR              null,
   city                 VARCHAR              null,
   district             VARCHAR              null,
   addr                 VARCHAR              null,
   train_item           VARCHAR              null,
   other_item           VARCHAR              null,
   field_type           VARCHAR              null,
   area                 FLOAT8               null,
   creator              INT8                 null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   updated_by           INT8                 null,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   domain_id            INT8                 null,
   addi                 JSONB                null,
   remark               VARCHAR              null,
   status               VARCHAR              null,
   constraint PK_T_INSURED_DETAIL primary key (id)
);

comment on table t_insured_detail is
'清单表，校车的校车信息 校车承运人存在同一行，但前端分开显示';

comment on column t_insured_detail.id is
'主键';

comment on column t_insured_detail.type is
'清单类别';

comment on column t_insured_detail.sub_type is
'清单子类型';

comment on column t_insured_detail.order_id is
'订单号（内部订单号)';

comment on column t_insured_detail.policy_id is
'保单号（在生成保单的时候才填写）';

comment on column t_insured_detail.name is
'姓名（比赛、教工、实习生、校车承运人）';

comment on column t_insured_detail.id_card_no is
'证件号码（比赛、教工、实习生)';

comment on column t_insured_detail.gender is
'性别 （比赛、教工、实习生';

comment on column t_insured_detail.birthday is
'出生日期（比赛、教工、实习生)';

comment on column t_insured_detail.role is
'职位(工作类型)（教工)';

comment on column t_insured_detail.org is
'所属机构（教工)';

comment on column t_insured_detail.class is
'班别（实习生)';

comment on column t_insured_detail.group_day is
'所在比赛/活动日期';

comment on column t_insured_detail.license_plate_no is
'车牌号码（校车信息 校车承运人)';

comment on column t_insured_detail.brand is
'厂牌类型（校车信息)';

comment on column t_insured_detail.driver_seat_number is
'司机座位（校车承运人）';

comment on column t_insured_detail.approved_passengers_num is
'核定客载人数';

comment on column t_insured_detail.seat_num is
'座位数（校车信息 校车承运人)';

comment on column t_insured_detail.road_grade is
'运营公路等级（校车信息)';

comment on column t_insured_detail.driver_license is
'驾驶证-图片';

comment on column t_insured_detail.driving_license is
'行驶证-图片（校车信息)';

comment on column t_insured_detail.action is
'修改类型：2:新增 4:删除 6:修改（此处需要对清单ID进行比对，才可以得出）';

comment on column t_insured_detail.err_msg is
'错误原因';

comment on column t_insured_detail.province is
'省';

comment on column t_insured_detail.city is
'市';

comment on column t_insured_detail.district is
'区';

comment on column t_insured_detail.addr is
'地址';

comment on column t_insured_detail.train_item is
'训练项目（英文逗号分隔）';

comment on column t_insured_detail.other_item is
'其它项目（英文逗号分隔）';

comment on column t_insured_detail.field_type is
'场地类型';

comment on column t_insured_detail.area is
'场地面积';

comment on column t_insured_detail.creator is
'创建者';

comment on column t_insured_detail.create_time is
'创建时间';

comment on column t_insured_detail.updated_by is
'更新者';

comment on column t_insured_detail.update_time is
'更新时间';

comment on column t_insured_detail.domain_id is
'数据隶属';

comment on column t_insured_detail.addi is
'附加信息';

comment on column t_insured_detail.remark is
'备注 （实习生)';

comment on column t_insured_detail.status is
'状态：0:有效 2:错误  4.拒保';

ALTER SEQUENCE t_insured_detail_id_seq RESTART WITH 20000;

/*==============================================================*/
/* Index: idx_insured_detail_o_p_n                              */
/*==============================================================*/
create unique index if not exists  idx_insured_detail_o_p_n on t_insured_detail (
id,
order_id,
policy_id,
name
);

/*==============================================================*/
/* Table: t_insured_terms                                       */
/*==============================================================*/
create table if not exists  t_insured_terms (
   id                   SERIAL not null,
   insurance_type_id    INT8                 null,
   topic                VARCHAR              null,
   parent_id            INT8                 null,
   level                INT2                 null,
   content              VARCHAR              null,
   updated_by           INT8                 null,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   creator              VARCHAR              null,
   domain_id            INT8                 null,
   addi                 JSONB                null,
   remark               VARCHAR              null,
   status               VARCHAR              null,
   constraint PK_T_INSURED_TERMS primary key (id)
);

comment on table t_insured_terms is
'保险条款';

comment on column t_insured_terms.id is
'编号';

comment on column t_insured_terms.insurance_type_id is
'保险类型';

comment on column t_insured_terms.topic is
'标题';

comment on column t_insured_terms.parent_id is
'父级ID';

comment on column t_insured_terms.level is
'级别';

comment on column t_insured_terms.content is
'内容';

comment on column t_insured_terms.updated_by is
'更新者';

comment on column t_insured_terms.update_time is
'更新时间';

comment on column t_insured_terms.create_time is
'创建时间';

comment on column t_insured_terms.creator is
'创建者账号';

comment on column t_insured_terms.domain_id is
'数据属主';

comment on column t_insured_terms.addi is
'附加数据';

comment on column t_insured_terms.remark is
'备注';

comment on column t_insured_terms.status is
'状态0:有效, 2:修改，4删除';

ALTER SEQUENCE t_insured_terms_id_seq RESTART WITH 20000;

/*==============================================================*/
/* Table: t_invigilation                                        */
/*==============================================================*/
create table if not exists  t_invigilation (
   id                   INT4                 not null,
   exam_session_id      INT8                 null,
   invigilator          INT8                 null,
   exam_room            INT8                 null,
   creator              INT8                 not null,
   create_time          TIMESTAMP            null,
   updated_by           INT8                 null,
   update_time          TIMESTAMP            null,
   addi                 JSONB                null,
   constraint PK_T_INVIGILATION primary key (id)
);

comment on table t_invigilation is
'监考安排表';

comment on column t_invigilation.id is
'编号';

comment on column t_invigilation.exam_session_id is
'考试场次id';

comment on column t_invigilation.invigilator is
'监考员id';

comment on column t_invigilation.exam_room is
'考场id';

comment on column t_invigilation.creator is
'创建者';

comment on column t_invigilation.create_time is
'创建时间';

comment on column t_invigilation.updated_by is
'更新者';

comment on column t_invigilation.update_time is
'更新时间';

comment on column t_invigilation.addi is
'附加信息';

/*==============================================================*/
/* Table: t_judge                                               */
/*==============================================================*/
create table if not exists  t_judge (
   id                   SERIAL not null,
   developer_id         INT8                 null,
   proof_id             INT8                 null,
   witness_id           INT8                 null,
   apply_time           INT8                 null,
   judge_time           INT8                 null,
   status               VARCHAR              null,
   constraint PK_T_JUDGE primary key (id)
);

comment on table t_judge is
'鉴定邀请表';

comment on column t_judge.id is
'评价编号';

comment on column t_judge.developer_id is
'被鉴定人';

comment on column t_judge.proof_id is
'鉴定项';

comment on column t_judge.witness_id is
'鉴定人';

comment on column t_judge.apply_time is
'申请鉴定时间';

comment on column t_judge.judge_time is
'鉴定时间';

comment on column t_judge.status is
'invited,已发出邀请
judged,已评价、签定
rejected,拒绝评价
expired,邀请已过期/拒绝评价';

/*==============================================================*/
/* Table: t_log                                                 */
/*==============================================================*/
create table if not exists  t_log (
   id                   SERIAL not null,
   grade                VARCHAR              null,
   msg                  VARCHAR              null,
   caller               VARCHAR              null,
   stacktrace           VARCHAR              null,
   namespace            VARCHAR              null,
   login_user_name      VARCHAR              null,
   login_user_id        INT8                 null,
   domain_id            INT8                 null,
   creator              INT8                 null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   constraint PK_T_LOG primary key (id)
);

comment on table t_log is
't_log';

comment on column t_log.id is
'编号';

comment on column t_log.grade is
'等级';

comment on column t_log.msg is
'消息';

comment on column t_log.caller is
'位置';

comment on column t_log.stacktrace is
'栈';

comment on column t_log.namespace is
'模块';

comment on column t_log.login_user_name is
'用户名';

comment on column t_log.login_user_id is
'用户编码';

comment on column t_log.domain_id is
'数据隶属';

comment on column t_log.creator is
'本数据创建者';

comment on column t_log.create_time is
'生成时间';

/*==============================================================*/
/* Index: t_log_PK                                              */
/*==============================================================*/
create unique index if not exists  t_log_PK on t_log (
id
);

/*==============================================================*/
/* Index: idx_t_log_base                                        */
/*==============================================================*/
create  index if not exists  idx_t_log_base on t_log (
grade,
create_time,
login_user_id
);

/*==============================================================*/
/* Table: t_mark                                                */
/*==============================================================*/
create table if not exists  t_mark (
   id                   INT4                 not null,
   teacher_id           INT8                 null,
   question_id          INT8                 null,
   examinee_id          INT8                 null,
   practice_submissions_id INT8                 null,
   practice_id          INT8                 null,
   exam_session_id      INT8                 null,
   mark_details         jsonb                null,
   status               VARCHAR(4)           null,
   addi                 jsonb                null,
   creator              INT8                 not null,
   create_time          INT8                 null,
   updated_by           INT8                 null,
   update_time          INT8                 null,
   score                double precision     null,
   constraint PK_T_MARK primary key (id)
);

comment on table t_mark is
'批改结果表';

comment on column t_mark.id is
'id';

comment on column t_mark.teacher_id is
'批阅教师id';

comment on column t_mark.question_id is
'题目id';

comment on column t_mark.examinee_id is
'批改的学生id';

comment on column t_mark.practice_submissions_id is
'练习提交id';

comment on column t_mark.practice_id is
'练习id';

comment on column t_mark.exam_session_id is
'考试场次id';

comment on column t_mark.mark_details is
'批改结果';

comment on column t_mark.status is
'状态 00:正常 02';

comment on column t_mark.addi is
'附加信息';

comment on column t_mark.creator is
'创建者';

comment on column t_mark.create_time is
'创建时间';

comment on column t_mark.updated_by is
'更新者';

comment on column t_mark.update_time is
'更新时间';

comment on column t_mark.score is
'分数';

/*==============================================================*/
/* Table: t_mark_info                                           */
/*==============================================================*/
create table if not exists  t_mark_info (
   id                   SERIAL               not null,
   exam_session_id      INT8                 null,
   practice_id          INT8                 null,
   mark_teacher_id      INT8                 null,
   mark_count           INT4                 null,
   mark_question_groups JSONB                null,
   mark_examinee_ids    JSONB                null,
   creator              INT8                 not null,
   create_time          INT8                 null,
   updated_by           INT8                 null,
   update_time          INT8                 null,
   addi                 JSONB                null,
   status               VARCHAR(8)           null,
   question_ids         jsonb                null,
   constraint PK_T_MARK_INFO primary key (id)
);

comment on table t_mark_info is
'批改信息配置表';

comment on column t_mark_info.id is
'id';

comment on column t_mark_info.exam_session_id is
'考试场次id';

comment on column t_mark_info.practice_id is
'练习id';

comment on column t_mark_info.mark_teacher_id is
'批改员id';

comment on column t_mark_info.mark_count is
'批改份数';

comment on column t_mark_info.mark_question_groups is
'批改题组';

comment on column t_mark_info.mark_examinee_ids is
'批改的学生数组';

comment on column t_mark_info.creator is
'创建者';

comment on column t_mark_info.create_time is
'创建时间';

comment on column t_mark_info.updated_by is
'更新者';

comment on column t_mark_info.update_time is
'更新时间';

comment on column t_mark_info.addi is
'附加信息';

comment on column t_mark_info.status is
'数据状态';

comment on column t_mark_info.question_ids is
'批改的题目集';

/*==============================================================*/
/* Table: t_mistake_correct                                     */
/*==============================================================*/
create table if not exists  t_mistake_correct (
   id                   SERIAL not null,
   order_id             INT8                 null,
   policyholder         JSONB                null,
   contact              JSONB                null,
   policyholder_id      INT8                 null,
   official_name_p      VARCHAR              null,
   id_card_type_p       VARCHAR              null,
   id_card_no_p         VARCHAR              null,
   gender_p             VARCHAR              null,
   birthday_p           INT8                 null,
   insured              JSONB                null,
   insured_id           INT8                 null,
   official_name        VARCHAR              null,
   id_card_type         VARCHAR              null,
   id_card_no           VARCHAR              null,
   gender               VARCHAR              null,
   birthday             INT8                 null,
   clear_list           BOOL                 null,
   insured_list         JSONB                null,
   insured_count        INT2                 null,
   commence_date        INT8                 null,
   expiry_date          INT8                 null,
   indate               INT8                 null,
   charge_mode          VARCHAR              null,
   modify_type          VARCHAR              null,
   activity_name        VARCHAR              null,
   activity_location    VARCHAR              null,
   activity_date_set    VARCHAR              null,
   insured_type         VARCHAR              null,
   schoolbus_company    VARCHAR              null,
   guarantee_item       JSONB                null,
   confirm_guarantee_star_time INT8                 null,
   non_compulsory_student_num INT8                 null,
   compulsory_student_num INT8                 null,
   dinner_num           INT8                 null,
   school_enrolment_total INT8                 null,
   shop_num             INT8                 null,
   canteen_num          INT8                 null,
   activity_desc        VARCHAR              null,
   invoice_header       VARCHAR              null,
   dispute_handling     VARCHAR              null,
   have_sudden_death    BOOL                 null,
   prev_policy_no       VARCHAR              null,
   revoked_policy_no    VARCHAR              null,
   pool_name            VARCHAR              null,
   have_explosive       BOOL                 null,
   have_rides           BOOL                 null,
   inner_area           FLOAT8               null,
   outer_area           FLOAT8               null,
   traffic_num          INT4                 null,
   temperature_type     VARCHAR              null,
   open_pool_num        INT2                 null,
   heated_pool_num      INT2                 null,
   training_pool_num    INT2                 null,
   pool_num             INT2                 null,
   custom_type          VARCHAR              null,
   same                 BOOL                 null,
   arbitral_agency      VARCHAR              null,
   endorsement_status   VARCHAR              null,
   application_files    JSONB                null,
   amount               FLOAT8               null,
   insured_group_by_day BOOL                 null,
   refused_reason       VARCHAR              null,
   pay_type             VARCHAR              null,
   need_balance         BOOL                 null,
   fee_scheme           JSONB                null,
   have_negotiated_price BOOL                 null,
   policy_scheme        JSONB                null,
   plan_id              INT8                 null,
   correct_level        VARCHAR              null,
   correct_log          JSONB                null,
   policy_regen         BOOL                 null,
   files                JSONB                null,
   files_to_remove      VARCHAR              null,
   creator              INT8                 null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   updated_by           INT8                 null,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   domain_id            INT8                 null,
   addi                 JSONB                null,
   remark               VARCHAR              null,
   status               VARCHAR              null,
   constraint PK_T_MISTAKE_CORRECT primary key (id)
);

comment on table t_mistake_correct is
'报错';

comment on column t_mistake_correct.id is
'id';

comment on column t_mistake_correct.order_id is
'order_id';

comment on column t_mistake_correct.policyholder is
'投保人';

comment on column t_mistake_correct.contact is
'投保联系人';

comment on column t_mistake_correct.policyholder_id is
'投保人编号';

comment on column t_mistake_correct.official_name_p is
'投保人姓名';

comment on column t_mistake_correct.id_card_type_p is
'投保人证件类型';

comment on column t_mistake_correct.id_card_no_p is
'投保人身份证号码';

comment on column t_mistake_correct.gender_p is
'投保人性别';

comment on column t_mistake_correct.birthday_p is
'投保人出生日期';

comment on column t_mistake_correct.insured is
'被保险人';

comment on column t_mistake_correct.insured_id is
'被保险人编号';

comment on column t_mistake_correct.official_name is
'姓名';

comment on column t_mistake_correct.id_card_type is
'证件类型';

comment on column t_mistake_correct.id_card_no is
'身份证号码';

comment on column t_mistake_correct.gender is
'性别';

comment on column t_mistake_correct.birthday is
'出生日期';

comment on column t_mistake_correct.clear_list is
'清除清单';

comment on column t_mistake_correct.insured_list is
'被保险人清单';

comment on column t_mistake_correct.insured_count is
'被保险人数';

comment on column t_mistake_correct.commence_date is
'起保日期';

comment on column t_mistake_correct.expiry_date is
'止保日期';

comment on column t_mistake_correct.indate is
'保险期间';

comment on column t_mistake_correct.charge_mode is
'计费方式';

comment on column t_mistake_correct.modify_type is
'修改类型：2:普通修改 4:修改发票抬头 6:增减被保险人';

comment on column t_mistake_correct.activity_name is
'比赛/活动名称';

comment on column t_mistake_correct.activity_location is
'比赛地点';

comment on column t_mistake_correct.activity_date_set is
'具体活动日期（英文逗号分隔多个日期）';

comment on column t_mistake_correct.insured_type is
'参赛人员类型（比赛，学生教师/成年人）';

comment on column t_mistake_correct.schoolbus_company is
'校车服务单位(校车)';

comment on column t_mistake_correct.guarantee_item is
'保障项目';

comment on column t_mistake_correct.confirm_guarantee_star_time is
'确认保障开始时间状态(校车, 食堂)';

comment on column t_mistake_correct.non_compulsory_student_num is
'非义务教育人数(校方)';

comment on column t_mistake_correct.compulsory_student_num is
'义务教育人数(校方)';

comment on column t_mistake_correct.dinner_num is
'就餐人数(食堂)';

comment on column t_mistake_correct.school_enrolment_total is
'注册学生人数(食堂)';

comment on column t_mistake_correct.shop_num is
'小卖铺数量(食堂)';

comment on column t_mistake_correct.canteen_num is
'食堂数量(食堂)';

comment on column t_mistake_correct.activity_desc is
'简述(比赛)';

comment on column t_mistake_correct.invoice_header is
'发票抬头';

comment on column t_mistake_correct.dispute_handling is
'争议处理';

comment on column t_mistake_correct.have_sudden_death is
'启用猝死责任险';

comment on column t_mistake_correct.prev_policy_no is
'续保保单号';

comment on column t_mistake_correct.revoked_policy_no is
'撤保保单号';

comment on column t_mistake_correct.pool_name is
'游泳池名称';

comment on column t_mistake_correct.have_explosive is
'危险易爆';

comment on column t_mistake_correct.have_rides is
'机械性游乐设施';

comment on column t_mistake_correct.inner_area is
'室内面积';

comment on column t_mistake_correct.outer_area is
'室外面积';

comment on column t_mistake_correct.traffic_num is
'每日客流量';

comment on column t_mistake_correct.temperature_type is
'常温池';

comment on column t_mistake_correct.open_pool_num is
'对外数量';

comment on column t_mistake_correct.heated_pool_num is
'恒温池数量';

comment on column t_mistake_correct.training_pool_num is
'训练池数量';

comment on column t_mistake_correct.pool_num is
'泳池数量';

comment on column t_mistake_correct.custom_type is
'场地使用性质:internal, open, both';

comment on column t_mistake_correct.same is
'被保险人同投保人';

comment on column t_mistake_correct.arbitral_agency is
'仲裁机构';

comment on column t_mistake_correct.endorsement_status is
'批单状态: 00未生成批单 04 已生成批改申请书 08用户上传批改申请书 12管理员上传批单';

comment on column t_mistake_correct.application_files is
'批改申请书';

comment on column t_mistake_correct.amount is
'更正后金额';

comment on column t_mistake_correct.insured_group_by_day is
'按天录入被保险人';

comment on column t_mistake_correct.refused_reason is
'拒绝理由';

comment on column t_mistake_correct.pay_type is
'支付方式: 对公转账/在线支付/线下支付';

comment on column t_mistake_correct.need_balance is
'需要录入差价';

comment on column t_mistake_correct.fee_scheme is
'计费标准/单价';

comment on column t_mistake_correct.have_negotiated_price is
'是否使用协议价';

comment on column t_mistake_correct.policy_scheme is
'保险方案';

comment on column t_mistake_correct.plan_id is
'plan_id';

comment on column t_mistake_correct.correct_level is
'更正等级';

comment on column t_mistake_correct.correct_log is
'更正记录';

comment on column t_mistake_correct.policy_regen is
'重新生成保单';

comment on column t_mistake_correct.files is
'附加文件';

comment on column t_mistake_correct.files_to_remove is
'待删除文件的digest,逗号分隔 ';

comment on column t_mistake_correct.creator is
'创建者用户ID';

comment on column t_mistake_correct.create_time is
'创建时间';

comment on column t_mistake_correct.updated_by is
'更新者';

comment on column t_mistake_correct.update_time is
'修改时间';

comment on column t_mistake_correct.domain_id is
'数据属主';

comment on column t_mistake_correct.addi is
'附加数据';

comment on column t_mistake_correct.remark is
'备注';

comment on column t_mistake_correct.status is
'状态,0: 草稿, 2: 受理中，4:同意， 6:拒绝';

ALTER SEQUENCE t_mistake_correct_id_seq RESTART WITH 20000;

/*==============================================================*/
/* Table: t_msg                                                 */
/*==============================================================*/
create table if not exists  t_msg (
   id                   SERIAL not null,
   sender               INT8                 null,
   target               JSONB                null,
   emit_type            VARCHAR              null,
   content              JSONB                not null,
   offline_target_list  JSONB                null,
   domain_id            INT8                 null,
   creator              INT8                 not null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   updated_by           INT8                 null,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   addi                 JSONB                null,
   remark               VARCHAR              null,
   status               VARCHAR              null default '01',
   constraint PK_T_MSG primary key (id)
);

comment on table t_msg is
'即时通信消息表
';

comment on column t_msg.id is
'参数编号';

comment on column t_msg.sender is
'消息发送者ID';

comment on column t_msg.target is
'消息接收者，JSON格式如下：
1. 用户／组示例
［
　{type:u,id:20004},
　{type:u,id:20005},
　{type:g,id:20005},
　{type:g,id:20008}
］

2. 所有用户示例
［{type:b}］


type, u:用户，g:组，b:广播(所有用户)，';

comment on column t_msg.emit_type is
'online: 仅在线用户, 忽略离线用户';

comment on column t_msg.content is
'消息内容';

comment on column t_msg.offline_target_list is
'未接收消息用户列表';

comment on column t_msg.domain_id is
'数据隶属';

comment on column t_msg.creator is
'本数据创建者';

comment on column t_msg.create_time is
'生成时间';

comment on column t_msg.updated_by is
'更新者';

comment on column t_msg.update_time is
'帐号信息更新时间';

comment on column t_msg.addi is
'附加信息';

comment on column t_msg.remark is
'备注';

comment on column t_msg.status is
'状态，00：草稿，01：有效，02：作废';

/*==============================================================*/
/* Table: t_msg_status                                          */
/*==============================================================*/
create table if not exists  t_msg_status (
   id                   SERIAL not null,
   msg_id               INT8                 not null,
   user_id              INT8                 not null,
   received_time        INT8                 null,
   viewed_time          INT8                 null,
   creator              INT8                 null,
   updated_by           INT8                 null,
   domain_id            INT8                 null,
   addi                 JSONB                null,
   remark               VARCHAR              null,
   status               VARCHAR              null,
   constraint PK_T_MSG_STATUS primary key (id)
);

comment on table t_msg_status is
'消息状态';

comment on column t_msg_status.id is
'参数编号';

comment on column t_msg_status.msg_id is
'消息编号';

comment on column t_msg_status.user_id is
'用户编号';

comment on column t_msg_status.received_time is
'用户接收消息时间';

comment on column t_msg_status.viewed_time is
'用户查看消息时间';

comment on column t_msg_status.creator is
'本数据创建者';

comment on column t_msg_status.updated_by is
'更新者';

comment on column t_msg_status.domain_id is
'数据隶属';

comment on column t_msg_status.addi is
'附加信息';

comment on column t_msg_status.remark is
'备注';

comment on column t_msg_status.status is
'状态，00：草稿，01：有效，02：作废';

/*==============================================================*/
/* Table: t_my_contact                                          */
/*==============================================================*/
create table if not exists  t_my_contact (
   id                   SERIAL not null,
   my_id                INT8                 not null,
   contact_type         VARCHAR              not null,
   contact_id           INT8                 not null,
   tag                  JSONB                null,
   creator              INT8                 null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   updated_by           INT8                 null,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   domain_id            INT8                 null,
   addi                 JSONB                null,
   remark               VARCHAR              null,
   status               VARCHAR              null default '01',
   constraint PK_T_MY_CONTACT primary key (id)
);

comment on table t_my_contact is
'聊天联系人， 可能是组或用户，参考微信通讯录设计';

comment on column t_my_contact.id is
'参数编号';

comment on column t_my_contact.my_id is
'联系人拥有者';

comment on column t_my_contact.contact_type is
'u: user, g: group';

comment on column t_my_contact.contact_id is
'联系人';

comment on column t_my_contact.tag is
'联系标签，json数组';

comment on column t_my_contact.creator is
'本数据创建者';

comment on column t_my_contact.create_time is
'生成时间';

comment on column t_my_contact.updated_by is
'更新者';

comment on column t_my_contact.update_time is
'更新时间';

comment on column t_my_contact.domain_id is
'数据隶属';

comment on column t_my_contact.addi is
'附加信息';

comment on column t_my_contact.remark is
'备注';

comment on column t_my_contact.status is
'状态，00：草稿，01：有效，02：作废';

/*==============================================================*/
/* Table: t_negotiated_price                                    */
/*==============================================================*/
create table if not exists  t_negotiated_price (
   id                   SERIAL not null,
   keyword              VARCHAR              null,
   commence_date        INT8                 null,
   location             VARCHAR              null,
   province             VARCHAR              null,
   city                 VARCHAR              null,
   district             VARCHAR              null,
   price_type           VARCHAR              not null,
   price                INT4                 not null,
   insurance_type_id    INT8                 null,
   match_times          INT4                 null default 0,
   indate               INT8                 null,
   creator              INT8                 null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   updated_by           INT8                 null,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   domain_id            INT8                 null,
   addi                 JSONB                null,
   remark               VARCHAR              null,
   status               VARCHAR              null,
   constraint PK_T_NEGOTIATED_PRICE primary key (id)
);

comment on table t_negotiated_price is
'协议价表';

comment on column t_negotiated_price.id is
'议价表设置编号';

comment on column t_negotiated_price.keyword is
'关键词';

comment on column t_negotiated_price.commence_date is
'开始日期（纳秒）';

comment on column t_negotiated_price.location is
'地点';

comment on column t_negotiated_price.province is
'省';

comment on column t_negotiated_price.city is
'市';

comment on column t_negotiated_price.district is
'区';

comment on column t_negotiated_price.price_type is
'议价类型（协议价/会议价）';

comment on column t_negotiated_price.price is
'议价价格';

comment on column t_negotiated_price.insurance_type_id is
'保险类型';

comment on column t_negotiated_price.match_times is
'匹配次数';

comment on column t_negotiated_price.indate is
'保险期间';

comment on column t_negotiated_price.creator is
'创建者用户ID';

comment on column t_negotiated_price.create_time is
'创建时间';

comment on column t_negotiated_price.updated_by is
'更新者';

comment on column t_negotiated_price.update_time is
'更新时间';

comment on column t_negotiated_price.domain_id is
'数据属主';

comment on column t_negotiated_price.addi is
'附加数据';

comment on column t_negotiated_price.remark is
'备注';

comment on column t_negotiated_price.status is
'0:有效, 2: 删除';

ALTER SEQUENCE t_negotiated_price_id_seq RESTART WITH 20000;

/*==============================================================*/
/* Index: idx_negotiation                                       */
/*==============================================================*/
create  index if not exists  idx_negotiation on t_negotiated_price (
commence_date,
location,
insurance_type_id
);

/*==============================================================*/
/* Table: t_order                                               */
/*==============================================================*/
create table if not exists  t_order (
   id                   SERIAL not null,
   trade_no             VARCHAR              null,
   pay_order_no         VARCHAR              null,
   transaction_id       VARCHAR              null,
   batch                VARCHAR              null,
   pay_time             INT8                 null,
   pay_type             VARCHAR              null,
   pay_channel          VARCHAR              null,
   pay_name             VARCHAR              null,
   pay_account_info     JSONB                null,
   refundable           BOOL                 null,
   unit_price           FLOAT8               not null,
   refund_desc          VARCHAR              null,
   amount               FLOAT8               not null,
   actual_amount        FLOAT8               null,
   balance              FLOAT8               null,
   balance_list         JSONB                null,
   insure_order_no      VARCHAR              null,
   refund_no            VARCHAR              null,
   refund               FLOAT8               null,
   refund_time          INT8                 null,
   confirm_refund       BOOL                 null,
   agency_id            INT8                 null,
   org_id               INT8                 not null,
   org_manager_id       INT8                 null,
   insurance_type       VARCHAR              not null,
   insurance_type_id    INT8                 null,
   insurance_police_id  INT8                 null,
   plan_id              INT8                 null,
   plan_name            VARCHAR              null,
   insurer              VARCHAR              null,
   policy_scheme        JSONB                null,
   policy_doc           VARCHAR              null,
   activity_name        VARCHAR              null,
   activity_category    VARCHAR              null,
   activity_desc        VARCHAR              null,
   activity_location    VARCHAR              null,
   activity_date_set    VARCHAR              null,
   copies_num           INT2                 null,
   insured_count        INT2                 null,
   compulsory_student_num INT8                 null,
   non_compulsory_student_num INT8                 null,
   contact              JSONB                null,
   fee_scheme           JSONB                null,
   car_service_target   VARCHAR              null,
   policyholder         JSONB                null,
   policyholder_type    VARCHAR              null,
   policyholder_id      INT8                 not null,
   same                 BOOL                 null,
   relation             VARCHAR              null,
   insured              JSONB                null,
   insured_id           INT8                 not null,
   health_survey        JSONB                null,
   org_name             VARCHAR              null,
   org_category         VARCHAR              null,
   have_insured_list    BOOL                 null,
   insured_group_by_day BOOL                 null,
   insured_type         VARCHAR              null,
   insured_list         JSONB                null,
   commence_date        INT8                 null,
   expiry_date          INT8                 null,
   indate               INT8                 null,
   sign                 VARCHAR              null,
   jurisdiction         VARCHAR              null,
   dispute_handling     VARCHAR              null,
   prev_policy_no       VARCHAR              null,
   insure_base          VARCHAR              null,
   blanket_insure_code  VARCHAR              null,
   custom_type          VARCHAR              null,
   train_projects       VARCHAR              null,
   business_locations   JSONB                null,
   open_pool_num        INT2                 null,
   heated_pool_num      INT2                 null,
   training_pool_num    INT2                 null,
   pool_num             INT2                 null,
   dinner_num           INT4                 null,
   have_dinner_num      BOOL                 null,
   canteen_num          INT4                 null,
   shop_num             INT4                 null,
   have_rides           BOOL                 null,
   have_explosive       BOOL                 null,
   inner_area           FLOAT8               null,
   outer_area           FLOAT8               null,
   pool_name            VARCHAR              null,
   arbitral_agency      VARCHAR              null,
   traffic_num          INT4                 null,
   temperature_type     VARCHAR              null,
   is_indoor            VARCHAR              null,
   extra                JSONB                null,
   bank_account         JSONB                null,
   pay_contact          VARCHAR              null,
   sudden_death_terms   VARCHAR              null,
   spec_agreement       VARCHAR              null,
   have_negotiated_price BOOL                 null,
   have_renewal_reminder BOOL                 null,
   lock_status          VARCHAR              null,
   insurance_company    VARCHAR              null,
   insurance_company_account VARCHAR              null,
   charge_mode          VARCHAR              null,
   can_revoke_order     BOOL                 null,
   can_public_transfers BOOL                 null,
   is_reminder          BOOL                 null default true,
   ground_num           INT2                 null,
   reminders_num        INT2                 null,
   reminder_times       VARCHAR              null,
   refused_reason       VARCHAR              null,
   unpaid_reason        VARCHAR              null,
   admin_received       BOOL                 null,
   user_received        BOOL                 null,
   have_sudden_death    BOOL                 null,
   have_confirm_date    BOOL                 null,
   is_invoice           BOOL                 null,
   inv_visible          VARCHAR              null,
   inv_borrow           VARCHAR              null,
   inv_title            VARCHAR              null,
   traits               VARCHAR              null,
   files                JSONB                null,
   inv_status           VARCHAR              null,
   order_status         VARCHAR              null,
   upd_status           VARCHAR              null,
   creator              INT8                 null,
   create_time          INT8                 not null default (extract(epoch from current_timestamp)*1000)::bigint,
   updated_by           INT8                 null,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   domain_id            INT8                 null,
   addi                 JSONB                null,
   remark               VARCHAR              null,
   status               VARCHAR              null,
   constraint PK_T_ORDER primary key (id)
);

comment on table t_order is
'订单';

comment on column t_order.id is
'订单编号';

comment on column t_order.trade_no is
'外部订单号';

comment on column t_order.pay_order_no is
'外部支付订单号';

comment on column t_order.transaction_id is
'支付平台订单号';

comment on column t_order.batch is
'批次编号';

comment on column t_order.pay_time is
'支付时间';

comment on column t_order.pay_type is
'支付方式: 对公转账/在线支付/线下支付';

comment on column t_order.pay_channel is
'校快保，泰合，近邻，人保，太平洋保险';

comment on column t_order.pay_name is
'支付方式名称';

comment on column t_order.pay_account_info is
'在线支付信息（支付后填写用于核查）';

comment on column t_order.refundable is
'是否支持在线退款';

comment on column t_order.unit_price is
'单价';

comment on column t_order.refund_desc is
'退款原因';

comment on column t_order.amount is
'应收金额';

comment on column t_order.actual_amount is
'实收金额';

comment on column t_order.balance is
'累计差额';

comment on column t_order.balance_list is
'差额详表';

comment on column t_order.insure_order_no is
'外部保险系统订单号';

comment on column t_order.refund_no is
'退款单号';

comment on column t_order.refund is
'(待)退款金额';

comment on column t_order.refund_time is
'退款时间';

comment on column t_order.confirm_refund is
'确认退款';

comment on column t_order.agency_id is
'表示用于统计的机构编号, 因为目前org_id,policyholder_id,insured_id都用来表示机构编号';

comment on column t_order.org_id is
'关联机构编号';

comment on column t_order.org_manager_id is
'关联机构管理人';

comment on column t_order.insurance_type is
'保险类型: 学生意外伤害险，活动/比赛险(旅游险),食品卫生责任险，教工责任险,校方责任险,实习生责任险,校车责任险,游泳池责任险';

comment on column t_order.insurance_type_id is
'保险产品编码，对应t_insurance_types.id';

comment on column t_order.insurance_police_id is
'保单编号[学意险]';

comment on column t_order.plan_id is
'保险方案编码，对应t_insurance_types.id';

comment on column t_order.plan_name is
'保险方案名称(前端暂存)';

comment on column t_order.insurer is
'保险方案承保公司(前端暂存)';

comment on column t_order.policy_scheme is
'保险方案';

comment on column t_order.policy_doc is
'保险条款';

comment on column t_order.activity_name is
'活动名称';

comment on column t_order.activity_category is
'活动类型';

comment on column t_order.activity_desc is
'活动描述';

comment on column t_order.activity_location is
'活动地点';

comment on column t_order.activity_date_set is
'具体活动日期(英文逗号隔开)';

comment on column t_order.copies_num is
'订单份数';

comment on column t_order.insured_count is
'总数量/保障人数/车辆数';

comment on column t_order.compulsory_student_num is
'义务教育学生人数（校方）';

comment on column t_order.non_compulsory_student_num is
'非义务教育人数（校方）';

comment on column t_order.contact is
'联系人';

comment on column t_order.fee_scheme is
'计费标准/单价';

comment on column t_order.car_service_target is
'校车服务对象';

comment on column t_order.policyholder is
'投保人';

comment on column t_order.policyholder_type is
'投保人类型：个人，机构';

comment on column t_order.policyholder_id is
'投保人编号';

comment on column t_order.same is
'投保人与被保险人是同一人';

comment on column t_order.relation is
'投保人与被保险人关系';

comment on column t_order.insured is
'被保险人';

comment on column t_order.insured_id is
'被保险人编号';

comment on column t_order.health_survey is
'健康调查结果';

comment on column t_order.org_name is
'学校名称，用户输入的需要新建的学校名称';

comment on column t_order.org_category is
'学校类别: 幼儿园、小学等';

comment on column t_order.have_insured_list is
'有被保险对象清单';

comment on column t_order.insured_group_by_day is
'被保险对象按日期分组';

comment on column t_order.insured_type is
'被保险人类型: 学生，非学生';

comment on column t_order.insured_list is
'被保险对象清单';

comment on column t_order.commence_date is
'起保日(毫秒)';

comment on column t_order.expiry_date is
'止保日(毫秒)';

comment on column t_order.indate is
'有效期(天)';

comment on column t_order.sign is
'用户签名';

comment on column t_order.jurisdiction is
'司法管辖权';

comment on column t_order.dispute_handling is
'争议处理';

comment on column t_order.prev_policy_no is
'续保保单号';

comment on column t_order.insure_base is
'承保基础';

comment on column t_order.blanket_insure_code is
'统保代码';

comment on column t_order.custom_type is
'场地使用性质:internal, open, both';

comment on column t_order.train_projects is
'训练项目';

comment on column t_order.business_locations is
'承保地址/区域范围/游泳池场地地址';

comment on column t_order.open_pool_num is
'对外开放游泳池数量';

comment on column t_order.heated_pool_num is
'恒温游泳池数量';

comment on column t_order.training_pool_num is
'培训游泳池数量';

comment on column t_order.pool_num is
'游泳池数量';

comment on column t_order.dinner_num is
'用餐人数';

comment on column t_order.have_dinner_num is
'是否开启就餐人数';

comment on column t_order.canteen_num is
'食堂个数';

comment on column t_order.shop_num is
'商店个数';

comment on column t_order.have_rides is
'营业场所是否有游泳池外游乐设施、机械性游乐设施等';

comment on column t_order.have_explosive is
'营业场所是否有制造、销售、储存易燃易爆危险品';

comment on column t_order.inner_area is
'室内面积';

comment on column t_order.outer_area is
'室外面积';

comment on column t_order.pool_name is
'游泳池名称(英文逗号分隔) ';

comment on column t_order.arbitral_agency is
'仲裁机构';

comment on column t_order.traffic_num is
'每日客流量（人）';

comment on column t_order.temperature_type is
'泳池性质:恒温、常温';

comment on column t_order.is_indoor is
'泳池特性:室内、室外';

comment on column t_order.extra is
'附加信息:
附加条款
企业经营描述
相关保险情况
保险公司提示
保险销售事项确认书
保险公司信息：经办人/工号、代理点代码、展业方式
产险销售人员：姓名、职业证号';

comment on column t_order.bank_account is
'对公帐号信息：户名、所在银行、账号';

comment on column t_order.pay_contact is
'线下支付联系人：微信二维码，base64';

comment on column t_order.sudden_death_terms is
'猝死条款内容：附加猝死保险责任每人限额5万元，累计限额5万元。附加猝死责任保险条款（经法院判决、仲裁机构裁决或根据县级以上政府及县级以上政府有关部门的行政决定书或者调解证明等材料，需由被保险人承担的经济赔偿责任，由保险人负责赔偿）';

comment on column t_order.spec_agreement is
'特别约定';

comment on column t_order.have_negotiated_price is
'是否使用协议价';

comment on column t_order.have_renewal_reminder is
'是否有续保通知';

comment on column t_order.lock_status is
'锁定状态:0(或留空):未锁定 2:未解锁 4:已解锁';

comment on column t_order.insurance_company is
'承保公司';

comment on column t_order.insurance_company_account is
'承保公司账号';

comment on column t_order.charge_mode is
'购买方式：按天购买，按月购买。学意险模式
	0: 团单, 特定起保、止保时间,
	2: 团单, 非12个保障时间，有起保时间
	4: 团单, 1年保障时间，指定起保日期
	6: 团单, 1年保障时间，不指定起保日期
	8: 散单, 1年保障时间，不指定起保日期';

comment on column t_order.can_revoke_order is
'是否允许撤销订单';

comment on column t_order.can_public_transfers is
'是否允许对公转账';

comment on column t_order.is_reminder is
'是否开启自动催款';

comment on column t_order.ground_num is
'场地个数';

comment on column t_order.reminders_num is
'催款次数';

comment on column t_order.reminder_times is
'催款时间,用英文逗号分隔';

comment on column t_order.refused_reason is
'拒保理由';

comment on column t_order.unpaid_reason is
'未收款理由';

comment on column t_order.admin_received is
'管理员已收件';

comment on column t_order.user_received is
'用户已收件';

comment on column t_order.have_sudden_death is
'是否开启猝死责任险';

comment on column t_order.have_confirm_date is
'是否确认保障时间';

comment on column t_order.is_invoice is
'是否开具发票, false|0: 未开具发票, true|1: 已开具发票';

comment on column t_order.inv_visible is
'用户是否可见发票, 0: 发票用户不可见, 2: 发票用户可见';

comment on column t_order.inv_borrow is
'00: 未生成发票, 30: 可预借发票, 34: 已生成预借发票申请函, 38: 用户已下载预借发票申请函, 42: 用户已上传盖章预借发票申请函';

comment on column t_order.inv_title is
'发票抬头修改状态, 16: 申请改发票抬头, 20: 已上传新发票, 24: 已下载新发票';

comment on column t_order.traits is
'特殊订单，标志为字符串数组，值域为: {"allowIntraday","ignoreAmountLimit","ignorePayDeadline","ignoreOrderDeadline","ignoreAgeLimit"},
allowIntraday:当天起保,ignoreAmountLimit:允许小于三人投保,ignorePayDeadline: 允许超时支付,ignoreOrderDeadline：允许超过截止时间投保,ignoreAgeLimit：允许超龄投保';

comment on column t_order.files is
'附加文件';

comment on column t_order.inv_status is
'发票状态, 00: 未生成发票, 04: 发票已上传, 08: 发票已下载, 12: 发票已快递';

comment on column t_order.order_status is
'订单状态, 00: 草稿, 04: 用户投保, 08: 申请议价, 12: 拒保, 16: 可支付, 18: 开始支付, 20: 已支付, 24: 退保, 28: 作废';

comment on column t_order.upd_status is
'订单更正状态, 00: 未更正, 02: 用户撤消申请, 04: 用户申请更正, 08: 接受更正,16: 更新订单, 20: 生成批改申请书, 24: 用户已下载申请书, 28: 用户已上传盖章批改申请书, 36: 管理员上传批改单, 40: 用户已下载批改单, 44: 批改单已快递给用户, 48: 申请被拒绝';

comment on column t_order.creator is
'创建者用户ID';

comment on column t_order.create_time is
'创建时间';

comment on column t_order.updated_by is
'更新者';

comment on column t_order.update_time is
'更新时间';

comment on column t_order.domain_id is
'数据属主';

comment on column t_order.addi is
'附加数据';

comment on column t_order.remark is
'备注';

comment on column t_order.status is
'0: 未支付, 2: 已支付，4: 已生成保单, 6: 已作废';

ALTER SEQUENCE t_order_id_seq RESTART WITH 20000;

-- alter table t_order add column agency_id int8;

drop trigger if exists trigger_order_agency_id on t_order;
drop function if exists order_agency_id_sync cascade;

create or replace function order_agency_id_sync()
returns trigger
as $$
-- declare agency_id int:=0;
begin

	case
	when TG_OP = 'INSERT' or TG_OP = 'UPDATE' then
		if new.org_id is not null and new.org_id > 0 then
			new.agency_id = new.org_id;
		elsif new.policyholder_id is not null and new.policyholder_id > 0 then
			new.agency_id = new.policyholder_id;
		elsif new.insured_id is not null and new.insured_id > 0 then
			new.agency_id = new.insured_id;
		else
			new.agency_id = -1;
		end if;
	end case;
    
	if new.insurer is null then
		select insurer into new.insurer 
		from v_insurer 
		where id=new.plan_id;
	end if;    
    
	return new;
end;
$$ language plpgsql;

create trigger trigger_order_agency_id before insert or update
on t_order
for each row
execute function order_agency_id_sync();

/*==============================================================*/
/* Index: idx_t_order_trade_no2                                 */
/*==============================================================*/
create unique index if not exists  idx_t_order_trade_no2 on t_order (
trade_no
);

/*==============================================================*/
/* Index: idx_t_order_create_time                               */
/*==============================================================*/
create  index if not exists  idx_t_order_create_time on t_order (
create_time
);

/*==============================================================*/
/* Index: idx_t_order_agency_id                                 */
/*==============================================================*/
create  index if not exists  idx_t_order_agency_id on t_order (
agency_id
);

/*==============================================================*/
/* Table: t_paper                                               */
/*==============================================================*/
create table if not exists  t_paper (
   id                   SERIAL               not null,
   domain_id            INT8                 not null,
   exampaper_id         INT8                 null,
   name                 VARCHAR(256)         null,
   assembly_type        VARCHAR(10)          null,
   category             VARCHAR(10)          null,
   level                VARCHAR(10)          null,
   suggested_duration   INT4                 null,
   description          TEXT                 null,
   tags                 JSONB                null,
   config               JSONB                null,
   creator              INT8                 null,
   create_time          INT8                 null,
   updated_by           INT8                 null,
   update_time          INT8                 null,
   version              INT8                 not null default 1,
   addi                 JSONB                null,
   status               VARCHAR(10)          null default '00',
   constraint PK_T_PAPER primary key (id)
);

comment on table t_paper is
'试卷';

comment on column t_paper.id is
'试卷ID';

comment on column t_paper.domain_id is
'所属域ID';

comment on column t_paper.exampaper_id is
'生成的考卷ID（当试卷已发布才会存在，存储试卷不可修改副本的ID值）';

comment on column t_paper.name is
'试卷名称';

comment on column t_paper.assembly_type is
'组卷方式 00：自定义组卷 02：随机组卷 04：智能刷题';

comment on column t_paper.category is
'试卷用途 00：考试 02：练习';

comment on column t_paper.level is
'试卷难度 00：简单 02：中等 04：困难';

comment on column t_paper.suggested_duration is
'建议时长，单位为分钟';

comment on column t_paper.description is
'试卷说明，介绍整张试卷';

comment on column t_paper.tags is
'试卷标签';

comment on column t_paper.config is
'随机组卷或智能刷题配置';

comment on column t_paper.creator is
'创建者';

comment on column t_paper.create_time is
'创建时间';

comment on column t_paper.updated_by is
'更新者';

comment on column t_paper.update_time is
'更新时间';

comment on column t_paper.version is
'版本号';

comment on column t_paper.addi is
'附加信息';

comment on column t_paper.status is
'状态 00：未发布， 02：已发布 04：作废 06：异常';

/*==============================================================*/
/* Table: t_paper_group                                         */
/*==============================================================*/
create table if not exists  t_paper_group (
   id                   SERIAL               not null,
   paper_id             INT8                 not null,
   name                 TEXT                 null,
   "order"              INT4                 null,
   creator              INT8                 null,
   create_time          INT8                 null,
   updated_by           INT8                 null,
   update_time          INT8                 null,
   addi                 JSONB                null,
   status               VARCHAR(10)          null default '00',
   constraint PK_T_PAPER_GROUP primary key (id)
);

comment on table t_paper_group is
'试卷题组';

comment on column t_paper_group.id is
'题组ID';

comment on column t_paper_group.paper_id is
'试卷ID';

comment on column t_paper_group.name is
'题组名称';

comment on column t_paper_group."order" is
'题组排序';

comment on column t_paper_group.creator is
'创建者';

comment on column t_paper_group.create_time is
'创建时间';

comment on column t_paper_group.updated_by is
'更新者';

comment on column t_paper_group.update_time is
'更新时间';

comment on column t_paper_group.addi is
'附加信息';

comment on column t_paper_group.status is
'状态 00：正常， 02：异常';

/*==============================================================*/
/* Table: t_paper_question                                      */
/*==============================================================*/
create table if not exists  t_paper_question (
   id                   SERIAL               not null,
   bank_question_id     INT8                 not null,
   "order"              INT4                 null,
   group_id             INT8                 null,
   score                FLOAT8               null,
   sub_score            JSONB                null,
   creator              INT8                 not null,
   create_time          INT8                 null,
   updated_by           INT8                 null,
   update_time          INT8                 null,
   addi                 JSONB                null,
   status               VARCHAR(10)          null default '00',
   constraint PK_T_PAPER_QUESTION primary key (id)
);

comment on table t_paper_question is
'试卷题目';

comment on column t_paper_question.id is
'试卷题目ID';

comment on column t_paper_question.bank_question_id is
'题库题目ID';

comment on column t_paper_question."order" is
'题目序号';

comment on column t_paper_question.group_id is
'所在题组ID';

comment on column t_paper_question.score is
'在试卷中的分值';

comment on column t_paper_question.sub_score is
'主观题小题分 [1,2,3] 分别存储主观题每小题分数';

comment on column t_paper_question.creator is
'创建者';

comment on column t_paper_question.create_time is
'创建时间';

comment on column t_paper_question.updated_by is
'更新者';

comment on column t_paper_question.update_time is
'更新时间';

comment on column t_paper_question.addi is
'附加信息';

comment on column t_paper_question.status is
'状态 00：正常 02：异常';

/*==============================================================*/
/* Table: t_param                                               */
/*==============================================================*/
create table if not exists  t_param (
   id                   SERIAL not null,
   belongto             INT8                 not null,
   name                 VARCHAR              not null,
   value                VARCHAR              null,
   data_type            VARCHAR              null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   creator              INT8                 null,
   domain_id            INT8                 null,
   addi                 JSONB                null,
   remark               VARCHAR              null,
   status               VARCHAR              null default '01',
   constraint PK_T_PARAM primary key (id)
);

comment on table t_param is
'提供用户设置参数
belongTo value scope
系统一级:1000-1990
系统二级:2000-2990
系统三级:3000-3990
系统四级:4000-4990

应用系统一级:11000-11990
应用系统二级:12000-12990
应用系统三级:13000-13990
应用系统四级:14000-14990
预置参数ID只使用偶数';

comment on column t_param.id is
'参数编号';

comment on column t_param.belongto is
'类属';

comment on column t_param.name is
'参数名称';

comment on column t_param.value is
'参数值';

comment on column t_param.data_type is
'数据类型, string,number,bool,nil';

comment on column t_param.create_time is
'生成时间';

comment on column t_param.creator is
'本数据创建者';

comment on column t_param.domain_id is
'数据隶属';

comment on column t_param.addi is
'附加信息';

comment on column t_param.remark is
'备注';

comment on column t_param.status is
'状态，00：草稿，01：有效，02：作废';

ALTER SEQUENCE t_param_id_seq RESTART WITH 20000;

/* belongTo value scope
系统一级:1000-1990
系统二级:2000-2990
系统三级:3000-3990
系统四级:4000-4990

应用系统一级:11000-11990
应用系统二级:12000-12990
应用系统三级:13000-13990
应用系统四级:14000-14990
预置参数ID只使用偶数  */


delete from t_param;

insert into t_param(id,belongto,name) values(11000,0,'校快保');

insert into t_param(id,belongto,name) values(12000,11000,'客户类型');

insert into t_param(id,belongto,name) values(12006,11000,'数据同步目标');
insert into t_param(id,belongto,name,value) values(13070,12006,'微信','0');
insert into t_param(id,belongto,name,value) values(13072,12006,'联保','2');
insert into t_param(id,belongto,name,value) values(13074,12006,'平安','4');

insert into t_param(id,belongto,name) values(12004,11000,'证件类型');
insert into t_param(id,belongto,name,value) values(13080,12004,'身份证','01');
insert into t_param(id,belongto,name,value) values(13082,12004,'护照','03');
insert into t_param(id,belongto,name,value) values(13084,12004,'港澳身份证','07');
insert into t_param(id,belongto,name,value) values(13086,12004,'赴台通行证','08');
insert into t_param(id,belongto,name,value) values(13088,12004,'港澳通行证','10');
insert into t_param(id,belongto,name,value) values(13090,12004,'外国人永久居留身份证','16');
insert into t_param(id,belongto,name,value) values(13092,12004,'台湾居民来往内地通行证','25');
insert into t_param(id,belongto,name,value) values(13094,12004,'其他','99');
insert into t_param(id,belongto,name,value) values(13096,12004,'军官证','99');


insert into t_param(id,belongto,name) values(12008,11000,'保险类型');
insert into t_param(id,belongto,name,value) values(13100,12008,'学生意外伤害险(期限一年起)','10000');
insert into t_param(id,belongto,name,value) values(13101,12008,'车辆险','1');
insert into t_param(id,belongto,name,value) values(13102,12008,'财产险','1');
insert into t_param(id,belongto,name,value) values(13104,12008,'旅游险','1');
insert into t_param(id,belongto,name,value) values(13106,12008,'比赛意外险','1');
insert into t_param(id,belongto,name,value) values(13108,12008,'校方（学生）责任险','1');
insert into t_param(id,belongto,name,value) values(13110,12008,'教职工责任险','1');
insert into t_param(id,belongto,name,value) values(13112,12008,'公众责任险','1');
insert into t_param(id,belongto,name,value) values(13114,12008,'其他','1');


insert into t_param(id,belongto,name) values(12010,11000,'治疗方式');
insert into t_param(id,belongto,name,value) values(13200,12010,'门诊','0');
insert into t_param(id,belongto,name,value) values(13202,12010,'住院','2');
insert into t_param(id,belongto,name,value) values(13204,12010,'门诊及住院','4');
insert into t_param(id,belongto,name,value) values(13206,12010,'其他','6');


insert into t_param(id,belongto,name) values(12012,11000,'性别');
insert into t_param(id,belongto,name,value) values(13230,12012,'男','1');
insert into t_param(id,belongto,name,value) values(13232,12012,'女','0');


insert into t_param(id,belongto,name) values(12014,11000,'与被保险人关系');
insert into t_param(id,belongto,name,value) values(13240,12014,'父子','0');
insert into t_param(id,belongto,name,value) values(13242,12014,'父女','2');
insert into t_param(id,belongto,name,value) values(13244,12014,'母子','4');
insert into t_param(id,belongto,name,value) values(13246,12014,'母女','6');
insert into t_param(id,belongto,name,value) values(13248,12014,'法定监护人','8');


insert into t_param(id,belongto,name) values(12016,11000,'账户类型');
insert into t_param(id,belongto,name,value) values(13258,12016,'本人','0');
insert into t_param(id,belongto,name,value) values(13260,12016,'父亲','0');
insert into t_param(id,belongto,name,value) values(13262,12016,'母亲','2');
insert into t_param(id,belongto,name,value) values(13264,12016,'其它监护人','4');
insert into t_param(id,belongto,name,value) values(13266,12016,'投保单位','6');



insert into t_param(id,belongto,name) values(12018,11000,'保险时间');
insert into t_param(id,belongto,name,value) values(13280,12018,'学平险默认止保日期[月-日]','08-31');
insert into t_param(id,belongto,name,value) values(13282,12018,'学平险默认起保日期[月-日]','09-01');
insert into t_param(id,belongto,name,value) values(13284,12018,'学平险默认投保起始日期[月-日]','09-01');
insert into t_param(id,belongto,name,value) values(13286,12018,'学平险默认投保结束日期[月-日]','09-30');
insert into t_param(id,belongto,name,value) values(13288,12018,'学平险默认投保时长[天]','30');
insert into t_param(id,belongto,name,value) values(13290,12018,'订单支付时限[秒],0表示不限制','900');


insert into t_param(id,belongto,name) values(12020,11000,'比赛/活动保险保险期间');
insert into t_param(id,belongto,name,value) values(13300,12020,'level1','30');
insert into t_param(id,belongto,name,value) values(13302,12020,'level2','61');
insert into t_param(id,belongto,name,value) values(13304,12020,'level3','92');
insert into t_param(id,belongto,name,value) values(13306,12020,'level4','182');
insert into t_param(id,belongto,name,value) values(13308,12020,'level5','364');


insert into t_param(id,belongto,name) values(12022,11000,'比赛/活动保险参数');
insert into t_param(id,belongto,name,value) values(13320,12022,'请求议价标准','5000');
insert into t_param(id,belongto,name,value) values(13322,12022,'活动简述最大长度','35');
insert into t_param(id,belongto,name,value) values(13324,12022,'比赛清单最低人数','3');
insert into t_param(id,belongto,name,value) values(13326,12022,'默认省份','广东省');
insert into t_param(id,belongto,name,value) values(13328,12022,'默认城市','广州市');
insert into t_param(id,belongto,name,value) values(13330,12022,'线下支付起步价','10000');


insert into t_param(id,belongto,name) values(12024,11000,'投保单位性质');
insert into t_param(id,belongto,name,value) values(13340,12024,'学校','0');
insert into t_param(id,belongto,name,value) values(13342,12024,'非学校','2');


insert into t_param(id,belongto,name) values(12026,11000,'投保联系人职位');
insert into t_param(id,belongto,name,value) values(13350,12026,'教师','0');
insert into t_param(id,belongto,name,value) values(13352,12026,'课组长','0');
insert into t_param(id,belongto,name,value) values(13354,12026,'行政','0');
insert into t_param(id,belongto,name,value) values(13356,12026,'家长/家委','0');
insert into t_param(id,belongto,name,value) values(13358,12026,'外聘教练','0');

insert into t_param(id,belongto,name,value) values(13360,12026,'工作人员','0');
insert into t_param(id,belongto,name,value) values(13362,12026,'校医','0');

insert into t_param(id,belongto,name,value) values(13370,12026,'主管','2');
insert into t_param(id,belongto,name,value) values(13372,12026,'经理','2');
insert into t_param(id,belongto,name,value) values(13374,12026,'其他','4');


insert into t_param(id,belongto,name) values(12030,11000,'比赛/活动保险参与人员类型');
insert into t_param(id,belongto,name,value) values(13380,12030,'学生/未成年人/教师','0');
insert into t_param(id,belongto,name,value) values(13382,12030,'成年人','2');


insert into t_param(id,belongto,name) values(12032,11000,'学校性质');
insert into t_param(id,belongto,name) values(13390,12032,'公办');
insert into t_param(id,belongto,name) values(13392,12032,'民办');
insert into t_param(id,belongto,name) values(13394,12032,'集体');
insert into t_param(id,belongto,name) values(13396,12032,'其他');


insert into t_param(id,belongto,name) values(12034,11000,'收费标准(校方)');
insert into t_param(id,belongto,name) values(13400,12034,'义务教育');
insert into t_param(id,belongto,name) values(13402,12034,'非义务教育');



-- name:订单状态 value:有该状态的险种
insert into t_param(id,belongto,name) values(12036,11000,'筛选器-订单状态');
insert into t_param(id,belongto,name,value) values(13410,12036,'未支付','');
insert into t_param(id,belongto,name,value) values(13412,12036,'未支付/待确认收款','10000');
insert into t_param(id,belongto,name,value) values(13414,12036,'待确认收款','10020,10060,10070,10080');
insert into t_param(id,belongto,name,value) values(13416,12036,'已支付','10000,10040');
insert into t_param(id,belongto,name,value) values(13418,12036,'交易成功','10020,10060,10070,10080');
insert into t_param(id,belongto,name,value) values(13420,12036,'已生成保单','10040');
insert into t_param(id,belongto,name,value) values(13422,12036,'待确认协议价','10000,10026');
insert into t_param(id,belongto,name,value) values(13424,12036,'催款超过5次仍未付款','10000,10060,10070,10080');
insert into t_param(id,belongto,name,value) values(13426,12036,'拒保','10000');
insert into t_param(id,belongto,name,value) values(13428,12036,'待确认差价','10000');
insert into t_param(id,belongto,name,value) values(13430,12036,'未解锁','10020,10060,10070,10080');
insert into t_param(id,belongto,name,value) values(13432,12036,'已解锁','10020,10060,10070,10080');
insert into t_param(id,belongto,name,value) values(13434,12036,'已退保','10040,10000,10020,10060,10070,10080');
insert into t_param(id,belongto,name,value) values(13436,12036,'草稿','10000,10020');

insert into t_param(id,belongto,name) values(12038,11000,'筛选器-保单状态');
insert into t_param(id,belongto,name,value) values(13450,12038,'未起保','10000,10040');
insert into t_param(id,belongto,name,value) values(13452,12038,'保障中','10000,10040');
insert into t_param(id,belongto,name,value) values(13454,12038,'已过期','10000,10040');
insert into t_param(id,belongto,name,value) values(13456,12038,'待确认修改','');


insert into t_param(id,belongto,name) values(12040,11000,'支付方式(校方)');
insert into t_param(id,belongto,name,value) values(13470,12040,'公对公转账',0);
insert into t_param(id,belongto,name,value) values(13472,12040,'私对公转账',2);
insert into t_param(id,belongto,name,value) values(13474,12040,'线下支付',4);
insert into t_param(id,belongto,name,value) values(13476,12040,'在线支付',6);


insert into t_param(id,belongto,name) values(12042,11000,'筛选器-学校类型');
insert into t_param(id,belongto,name,value,remark) values(13480,12042,'幼儿园','10040,10022,10024,10028,10030','非义务教育');
insert into t_param(id,belongto,name,value,remark) values(13482,12042,'小学','10040,10022,10024,10028,10030','义务教育');
insert into t_param(id,belongto,name,value,remark) values(13484,12042,'初中','10040,10022,10024,10028,10030','义务教育');
insert into t_param(id,belongto,name,value,remark) values(13486,12042,'高中','10040,10022,10024,10028,10030','非义务教育');
insert into t_param(id,belongto,name,value,remark) values(13488,12042,'完中','10040,10022,10024,10028,10030','义务教育,非义务教育');
insert into t_param(id,belongto,name,value,remark) values(13490,12042,'九年一贯制','10022,10024,10028,10030','义务教育');
insert into t_param(id,belongto,name,value,remark) values(13492,12042,'高职/大专','10022,10024,10026,10028,10030','非义务教育');
insert into t_param(id,belongto,name,value,remark) values(13494,12042,'大学','10040,10022,10024,10026,10028,10030','非义务教育');
insert into t_param(id,belongto,name,value,remark) values(13496,12042,'其他','10022,10024,10028,10030','义务教育,非义务教育');


insert into t_param(id,belongto,name) values(12044,11000,'筛选器-缴费状态');
insert into t_param(id,belongto,name,value) values(13500,12044,'未缴费','0');
insert into t_param(id,belongto,name,value) values(13502,12044,'已缴费','0');


insert into t_param(id,belongto,name) values(12046,11000,'筛选器-付款方式');
insert into t_param(id,belongto,name,value) values(13510,12046,'在线支付','0');
insert into t_param(id,belongto,name,value) values(13512,12046,'对公转账','2');


insert into t_param(id,belongto,name) values(12048,11000,'地区选择器默认值');
insert into t_param(id,belongto,name,value) values(13520,12048,'默认省份','广东省');
insert into t_param(id,belongto,name,value) values(13522,12048,'默认城市','广州市');
insert into t_param(id,belongto,name,value) values(13524,12048,'默认地区','天河区');


insert into t_param(id,belongto,name) values(12050,11000,'文件标签');
insert into t_param(id,belongto,name,value) values(13530,12050,'附件','0');
insert into t_param(id,belongto,name,value) values(13532,12050,'投保单盖章扫描件','2');
insert into t_param(id,belongto,name,value) values(13534,12050,'投保清单盖章扫描件','4');
insert into t_param(id,belongto,name,value) values(13536,12050,'付款凭证回执','6');
insert into t_param(id,belongto,name,value) values(13538,12050,'转账授权说明盖章扫描件','8');


insert into t_param(id,belongto,name) values(12052,11000,'餐饮场所责任保险子类别');
insert into t_param(id,belongto,name,value) values(13550,12052,'食堂','0');
insert into t_param(id,belongto,name,value) values(13552,12052,'小卖铺','2');
insert into t_param(id,belongto,name,value) values(13554,12052,'食堂+小卖铺','4');


insert into t_param(id,belongto,name) values(12054,11000,'争议处理');
insert into t_param(id,belongto,name,value) values(13560,12054,'诉讼','0');
insert into t_param(id,belongto,name,value) values(13562,12054,'仲裁','2');


insert into t_param(id,belongto,name) values(12056,11000,'支付方式(太平洋)');
insert into t_param(id,belongto,name,value) values(13570,12056,'公对公转账','0');
insert into t_param(id,belongto,name,value) values(13572,12056,'扫码支付','2');


insert into t_param(id,belongto,name) values(12058,11000,'是否首次投保');
insert into t_param(id,belongto,name,value) values(13580,12058,'新单','0');
insert into t_param(id,belongto,name,value) values(13582,12058,'续保','2');


insert into t_param(id,belongto,name) values(12060,11000,'俱乐部/场地责任保险子类别');
insert into t_param(id,belongto,name,value) values(13590,12060,'俱乐部','0');
insert into t_param(id,belongto,name,value) values(13592,12060,'场地','2');


insert into t_param(id,belongto,name) values(12062,11000,'营业性质');
insert into t_param(id,belongto,name,value) values(13600,12062,'文体体育','0');
insert into t_param(id,belongto,name,value) values(13602,12062,'广告','2');
insert into t_param(id,belongto,name,value) values(13604,12062,'事业单位','4');
insert into t_param(id,belongto,name,value) values(13606,12062,'政府机关','6');
insert into t_param(id,belongto,name,value) values(13608,12062,'其它','8');


insert into t_param(id,belongto,name) values(12064,11000,'场地使用性质');
insert into t_param(id,belongto,name,value) values(13620,12064,'本公司/俱乐部会员的训练时间段','0');
insert into t_param(id,belongto,name,value) values(13622,12064,'对公众开放','2');
insert into t_param(id,belongto,name,value) values(13624,12064,'两者皆有','4');


insert into t_param(id,belongto,name) values(12066,11000,'泳池性质');
insert into t_param(id,belongto,name,value) values(13630,12066,'恒温','0');
insert into t_param(id,belongto,name,value) values(13632,12066,'常温','2');
insert into t_param(id,belongto,name,value) values(13634,12066,'恒温/常温','4');

insert into t_param(id,belongto,name) values(12068,11000,'比赛/活动组织方责任险保险期间');
insert into t_param(id,belongto,name,value) values(13640,12068,'level1','30');
insert into t_param(id,belongto,name,value) values(13642,12068,'level2','61');
insert into t_param(id,belongto,name,value) values(13644,12068,'level3','92');
insert into t_param(id,belongto,name,value) values(13646,12068,'level4','182');
insert into t_param(id,belongto,name,value) values(13648,12068,'level5','365');


insert into t_param(id,belongto,name) values(12070,11000,'文件路径');
insert into t_param(id,belongto,name,value) values(13660,12070,'预借发票申请函','预借发票申请函.docx');
insert into t_param(id,belongto,name,value) values(13662,12070,'转账授权说明','授权转账说明.docx');


insert into t_param(id,belongto,name) values(12072,11000,'议价类型');
insert into t_param(id,belongto,name,value) values(13680,12072,'协议价','0');
insert into t_param(id,belongto,name,value) values(13682,12072,'会议价','2');

insert into t_param(id,belongto,name) values(12074,11000,'训练项目');
insert into t_param(id,belongto,name,value) values(13690,12074,'足球','0');
insert into t_param(id,belongto,name,value) values(13692,12074,'篮球','2');
insert into t_param(id,belongto,name,value) values(13694,12074,'乒乓球','4');
insert into t_param(id,belongto,name,value) values(13696,12074,'羽毛球','6');
insert into t_param(id,belongto,name,value) values(13698,12074,'健美操','8');
insert into t_param(id,belongto,name,value) values(13700,12074,'拉丁舞','10');
insert into t_param(id,belongto,name,value) values(13702,12074,'滑轮','12');
insert into t_param(id,belongto,name,value) values(13704,12074,'三棋','14');
insert into t_param(id,belongto,name,value) values(13706,12074,'跆拳道','16');
insert into t_param(id,belongto,name,value) values(13708,12074,'田径','18');
insert into t_param(id,belongto,name,value) values(13710,12074,'跳绳','20');
insert into t_param(id,belongto,name,value) values(13712,12074,'网球','22');
insert into t_param(id,belongto,name,value) values(13714,12074,'无人机','24');
insert into t_param(id,belongto,name,value) values(13716,12074,'武术','26');
insert into t_param(id,belongto,name,value) values(13718,12074,'平衡车','28');

insert into t_param(id,belongto,name) values(12076,11000,'场地类型');
insert into t_param(id,belongto,name,value) values(13730,12076,'对外开放场地','0');
insert into t_param(id,belongto,name,value) values(13732,12076,'学员培训地点','2');
insert into t_param(id,belongto,name,value) values(13734,12076,'对外开放和培训地点','4');


insert into t_param(id,belongto,name) values(12078,11000,'联系客服');
insert into t_param(id,belongto,name,value) values(13740,12078,'比赛保险(罗先生)','13710833615');
insert into t_param(id,belongto,name,value) values(13742,12078,'学意校责(周小姐)','13925001114');
insert into t_param(id,belongto,name,value) values(13744,12078,'监督电话(蔡小姐)','13925009308');


insert into t_param(id,belongto,name) values(12080,11000,'治疗结果');
insert into t_param(id,belongto,name,value) values(13760,12080,'治愈','0');
insert into t_param(id,belongto,name,value) values(13762,12080,'残疾','2');
insert into t_param(id,belongto,name,value) values(13764,12080,'死亡','4');
insert into t_param(id,belongto,name,value) values(13766,12080,'其他','6');


insert into t_param(id,belongto,name) values(12082,11000,'出险原因');
insert into t_param(id,belongto,name,value) values(13780,12082,'意外伤害','0');
insert into t_param(id,belongto,name,value) values(13782,12082,'疾病住院（满24小时）','2');

insert into t_param(id,belongto,name) values(12084,11000,'教职员工职位');
insert into t_param(id,belongto,name,value) values(13800,12084,'教师','0');
insert into t_param(id,belongto,name,value) values(13802,12084,'老师','2');
insert into t_param(id,belongto,name,value) values(13804,12084,'保育员','4');
insert into t_param(id,belongto,name,value) values(13806,12084,'厨师','6');
insert into t_param(id,belongto,name,value) values(13808,12084,'后厨帮工','8');
insert into t_param(id,belongto,name,value) values(13810,12084,'保洁员','10');
insert into t_param(id,belongto,name,value) values(13812,12084,'保安','12');
insert into t_param(id,belongto,name,value) values(13814,12084,'后勤人员','14');
insert into t_param(id,belongto,name,value) values(13816,12084,'校长','16');
insert into t_param(id,belongto,name,value) values(13818,12084,'园长','18');
insert into t_param(id,belongto,name,value) values(13820,12084,'财务','20');
insert into t_param(id,belongto,name,value) values(13822,12084,'实习教师','22');


insert into t_param(id,belongto,name) values(12086,11000,'更正状态');
insert into t_param(id,belongto,name,value) values(13900,12086,'更正申请','2');
insert into t_param(id,belongto,name,value) values(13902,12086,'同意更正','4');
insert into t_param(id,belongto,name,value) values(13904,12086,'拒绝更正','6');

insert into t_param(id,belongto,name) values(12088,11000,'更正类型');
insert into t_param(id,belongto,name,value) values(13910,12088,'管理员更正','0');
insert into t_param(id,belongto,name,value) values(13912,12088,'普通更正','2');
insert into t_param(id,belongto,name,value) values(13914,12088,'发票抬头修改','4');
insert into t_param(id,belongto,name,value) values(13916,12088,'增减被保险人','6');


insert into t_param(id,belongto,name) values(12090,11000,'智能客服');
insert into t_param(id,belongto,name,value) values(13930,12090,'联系电话','13925001114');

insert into t_param(id,belongto,name,addi) values(14000,11000,'健康告知书',
'{"title":"健康告知","tips":"特别提示:投保人有如实告知的法定义务，如对询问事项不如实告知，保险公司有权依法解除合同，且对保险合同解除前发生的保险事故，保险公司不承担保险责任，敬请如实告知！","signTitle":"被保险人/监护人亲笔签名","question":[{"SN":1,"question":"近1年有无因患重大疾病接受医师治疗、诊疗或用药？或被建议治疗、住院或手术？有无休病假","type":"radio","label":["有","无"],"answer":"","desc":null,"descWhen":["有"],"descLabel":"如有，请描述"},{"SN":2,"question":"过去5年有无因同一疾病，多次就诊，或持续存在的异常体征？或因病住院治疗七日以上","type":"radio","label":["有","无"],"answer":"","desc":null,"descWhen":["有"],"descLabel":"如有，请描述"},{"SN":3,"question":"有无先天性、遗传性、精神性疾病？或身体有无畸形、残疾、残障状况","type":"radio","label":["有","无"],"answer":"","desc":null,"descWhen":["有"],"descLabel":"如有，请描述"},{"SN":4,"question":"有无家族病史（如：父母、兄弟姐妹、子女）","type":"radio","label":["有","无"],"answer":"","desc":null,"descWhen":["有"],"descLabel":"如有，请描述"},{"SN":5,"question":"现在或过去有无患过小儿麻痹、儿童多动症、麻疹、癫痫、尿毒症、白血病等疾病","type":"radio","label":["有","无"],"answer":"","desc":null,"descWhen":["有"],"descLabel":"如有，请描述"},{"SN":6,"question":"现在或过去有无患呼吸系统、神经系统、消化系统、免疫系统等系统方面的异常、失能或疾病","type":"radio","label":["有","无"],"answer":"","desc":null,"descWhen":["有"],"descLabel":"如有，请描述"},{"SN":7,"question":"目前身体有无不适症状？或有心血管系统疾病或症状？","type":"radio","label":["有","无"],"answer":"","desc":null,"descWhen":["有"],"descLabel":"如有，请描述"},{"SN":8,"question":"女性告知事项(男性无需填写)有无乳房异常症状或疾病、生殖器官异常症状或疾病、宫颈涂片检查不正常","type":"radio","label":["有","无"],"answer":"","desc":null,"descWhen":["有"],"descLabel":"如有，请描述"}]}');

CREATE OR REPLACE FUNCTION createSchoolLayout(
  schoolType varchar,
  rootID int DEFAULT 12000,

  gradeBegin int DEFAULT 1,
  gradeCount int DEFAULT 3,
  gradePrefix varchar DEFAULT '',
  gradeSuffix varchar DEFAULT '年级',

  classBegin int DEFAULT 1,
  classCount int DEFAULT 30,
  classZero bool DEFAULT TRUE,

  classPrefix varchar DEFAULT '',
  classSuffix varchar DEFAULT '班',
  classNEC bool DEFAULT TRUE
) returns bool as $$
DECLARE
countNumber CONSTANT VARCHAR[]:='{"一","二","三","四","五","六","七","八","九","十","零"}';
grade_ VARCHAR;
class_ VARCHAR;
layName VARCHAR;
firstLayName VARCHAR;
secondLayName VARCHAR;
tenPos int;
unitPos int;
tempID int;
schoolTypeID int;
gradeID int;
totalClassCount int;
BEGIN
    totalClassCount := classCount + 1;
  if classNEC then
       totalClassCount := classCount + 1;
    end if;


  insert into t_param(belongto,name,data_type,addi) values(rootID,schoolType,'string',cast('{"gradeCount":'||gradeCount||',"classCount":'|| totalClassCount ||'}' as jsonb));
  select currval('t_param_id_seq') into schoolTypeID;

    raise info 'rootID: %, schoolType: %, schoolTypeID: %',rootID,schoolType,schoolTypeID;
  FOR i in  gradeBegin .. (gradeBegin + gradeCount - 1) LOOP
    if i <= 10 then
      layName:=countNumber[i];
    elsif i < 100 then
      tenPos:=i/10;
      unitPos:=i%10;
      firstLayName:=countNumber[tenPos];
      secondLayName:=countNumber[unitPos];
      if tenPos = 1 then
        firstLayName:='';
      end if;
      if unitPos = 0 then
        secondLayName:='';
      end if;
      layName:=firstLayName || '十' || secondLayName;
    else
      RAISE EXCEPTION 'unsupport number greater then 99';
    end if;
    
    grade_:=gradePrefix || layName || gradeSuffix;
        

    insert into t_param(belongto,name,data_type,addi) values(schoolTypeID,grade_,'string',cast('{"SN":'||i||'}' as jsonb));
    select currval('t_param_id_seq') into gradeID;
    raise info 'belongto: %, grade: %, gradeID: %',schoolTypeID,grade_,gradeID;

        if classNEC then
          class_:= '未分班';
      insert into t_param(belongto,name,data_type,addi) values(gradeID,class_,'string','{"SN":-1}');
            raise info 'gradeID: %, class_: %',gradeID,class_;
        end if; 
        
    if classZero then
      class_:=classPrefix || countNumber[11] || classSuffix;
      insert into t_param(belongto,name,data_type,addi) values(gradeID,class_,'string','{"SN":0}');
            raise info 'gradeID: %, class_: %',gradeID,class_;          
    end if;

    
    FOR j in classBegin .. (classBegin + classCount - 1) LOOP
      if j <= 10 then
        layName:=countNumber[j];
      elsif j < 100 then
        tenPos:=j/10;
        unitPos:=j%10;
        firstLayName:=countNumber[tenPos];
        secondLayName:=countNumber[unitPos];
        if tenPos = 1 then
          firstLayName:='';
        end if;
        if unitPos = 0 then
          secondLayName:='';
        end if;
        layName:=firstLayName || '十' || secondLayName;
      else
        RAISE EXCEPTION 'unsupport number greater then 99';
      end if;
      class_:=classPrefix || layName || classSuffix;
      insert into t_param(belongto,name,data_type,addi) values(gradeID,class_,'string',cast('{"SN":'||j||'}' as jsonb));
            raise info 'belongto: %, name: %, ',gradeID,class_;
    END LOOP;

    class_:= '其它';
    
    insert into t_param(belongto,name,data_type,addi) values(gradeID,class_,'string',cast('{"SN":'|| classBegin + classCount||'}' as jsonb));
    select currval('t_param_id_seq') into tempID;
    raise info 'id: %, belongto: %, name: %, ',tempID,gradeID,class_;
    
  END LOOP;

  RETURN TRUE;
END;
$$
LANGUAGE PLPGSQL;

select createSchoolLayout(
  rootID=>12000,
  schoolType=>'幼儿园',

  gradeBegin=>1,
  gradeCount=>4,
  gradePrefix=>'',
  gradeSuffix=>'年级',

  classBegin=>1,
  classCount=>30,
  classZero=>false,

  classPrefix=>'',
  classSuffix=>'班'
);

update t_param set value='小小班' where id =(select id from t_param where belongto=(
  select id from t_param where belongto=12000 and name='幼儿园'
) and name='一年级');

update t_param set value='小班' where id =(select id from t_param where belongto=(
  select id from t_param where belongto=12000 and name='幼儿园'
) and name='二年级');



update t_param set value='中班' where id =(select id from t_param where belongto=(
  select id from t_param where belongto=12000 and name='幼儿园'
) and name='三年级');


update t_param set value='大班' where id =(select id from t_param where belongto=(
  select id from t_param where belongto=12000 and name='幼儿园'
) and name='四年级');


select createSchoolLayout(
  rootID=>12000,
  schoolType=>'小学',
  gradeBegin=>1,
  gradeCount=>6,
  gradePrefix=>'',
  gradeSuffix=>'年级',

  classBegin=>1,
  classCount=>30,
  classZero=>false,

  classPrefix=>'',
  classSuffix=>'班'
);

select createSchoolLayout(
  rootID=>12000,
  schoolType=>'初中',
  gradeBegin=>1,
  gradeCount=>3,
  gradePrefix=>'初',
  gradeSuffix=>'',
  classBegin=>1,
  classCount=>30,
  classZero=>true,
  classPrefix=>'',
  classSuffix=>'班'
);

select createSchoolLayout(
  rootID=>12000,
  schoolType=>'高中',
  gradeBegin=>1,
  gradeCount=>3,
  gradePrefix=>'高',
  gradeSuffix=>'',
  classBegin=>1,
  classCount=>30,
  classZero=>true,
  classPrefix=>'',
  classSuffix=>'班'
);

select createSchoolLayout(
  rootID=>12000,
  schoolType=>'九年一贯制',
  gradeBegin=>1,
  gradeCount=>9,
  gradePrefix=>'',
  gradeSuffix=>'年级',
  classBegin=>1,
  classCount=>30,
  classZero=>true,
  classPrefix=>'',
  classSuffix=>'班'
);

update t_param set value='初一' where id =(select id from t_param where belongto=(
  select id from t_param where belongto=12000 and name='九年一贯制'
) and name='七年级');

update t_param set value='初二' where id =(select id from t_param where belongto=(
  select id from t_param where belongto=12000 and name='九年一贯制'
) and name='八年级');

update t_param set value='初三' where id =(select id from t_param where belongto=(
  select id from t_param where belongto=12000 and name='九年一贯制'
) and name='九年级');

select createSchoolLayout(
  rootID=>12000,
  schoolType=>'完中',

    gradeBegin=>1,
  gradeCount=>6,
  gradePrefix=>'初',
  gradeSuffix=>'',

  classBegin=>1,
  classCount=>30,
  classZero=>true,

  classPrefix=>'',
  classSuffix=>'班'
);


update t_param set name='高一'
where 
    id =(select id from t_param where belongto=(select id from t_param where belongto=12000 and name='完中') and name='初四');

update t_param set name='高二'
where 
    id =(select id from t_param where belongto=(select id from t_param where belongto=12000 and name='完中') and name='初五');

update t_param set name='高三'
where 
    id =(select id from t_param where belongto=(select id from t_param where belongto=12000 and name='完中') and name='初六');

/*
select createSchoolLayout(
  rootID=>12000,
  schoolType=>'中职',
  gradeBegin=>1,
  gradeCount=>3,
  gradePrefix=>'',
  gradeSuffix=>'年级',
  classBegin=>1,
  classCount=>16,
  classZero=>false,
  classPrefix=>'',
  classNEC=>false,
  classSuffix=>'班'
);
*/

select createSchoolLayout(
  rootID=>12000,
  schoolType=>'高职',
  gradeBegin=>1,
  gradeCount=>3,
  gradePrefix=>'大',
  gradeSuffix=>'',
  classBegin=>1,
  classCount=>16,
  classZero=>false,
  classPrefix=>'',
classNEC=>false,  
  classSuffix=>'班'
);


select createSchoolLayout(
  rootID=>12000,
  schoolType=>'大学',
  gradeBegin=>1,
  gradeCount=>4,
  gradePrefix=>'大',
  gradeSuffix=>'',
  classBegin=>1,
  classCount=>30,
  classZero=>false,
  classPrefix=>'',
  classNEC=>false,
  classSuffix=>'班'
);

/*
select createSchoolLayout(
  rootID=>12000,
  schoolType=>'研究生院',
  gradeBegin=>1,
  gradeCount=>3,
  gradePrefix=>'研',
  gradeSuffix=>'',
  classBegin=>1,
  classCount=>16,
  classZero=>false,
  classPrefix=>'',
  classNEC=>false,
  classSuffix=>'班'
);
*/








/*==============================================================*/
/* Index: t_param_full_idx                                      */
/*==============================================================*/
create unique index if not exists  t_param_full_idx on t_param (
belongto,
name
);

/*==============================================================*/
/* Table: t_pay_account                                         */
/*==============================================================*/
create table if not exists  t_pay_account (
   id                   SERIAL not null,
   type                 VARCHAR              null,
   name                 VARCHAR              null,
   app_id               VARCHAR              null,
   account              VARCHAR              null,
   key                  VARCHAR              null,
   cert                 VARCHAR              null,
   refundable           BOOL                 null,
   updated_by           INT8                 null,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   creator              INT8                 null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   domain_id            INT8                 null,
   addi                 JSONB                null,
   remark               VARCHAR              null,
   status               VARCHAR              null default '02',
   constraint PK_T_PAY_ACCOUNT primary key (id)
);

comment on table t_pay_account is
'支付账号信息';

comment on column t_pay_account.id is
'编号';

comment on column t_pay_account.type is
'wx_mp: 微信公众号, wx_open: 微信开放平台, ali: 阿里, ui: 联保';

comment on column t_pay_account.name is
'名称：校快保，泰合，联保，近邻，人保，太平洋保险，人寿';

comment on column t_pay_account.app_id is
'关联应用ID: 微信公众号';

comment on column t_pay_account.account is
'账号，微信支付商户号，支付宝商户号';

comment on column t_pay_account.key is
'密钥';

comment on column t_pay_account.cert is
'证书';

comment on column t_pay_account.refundable is
'是否支持退款';

comment on column t_pay_account.updated_by is
'更新者';

comment on column t_pay_account.update_time is
'帐号信息更新时间';

comment on column t_pay_account.creator is
'本数据创建者';

comment on column t_pay_account.create_time is
'生成时间';

comment on column t_pay_account.domain_id is
'数据隶属';

comment on column t_pay_account.addi is
'附加信息';

comment on column t_pay_account.remark is
'备注';

comment on column t_pay_account.status is
'状态，00：草稿，02：有效，04: 停用，06：作废';

ALTER SEQUENCE t_pay_account_id_seq RESTART WITH 20000;


/*==============================================================*/
/* Index: account_info                                          */
/*==============================================================*/
create unique index if not exists  account_info on t_pay_account (
type,
account,
app_id
);

/*==============================================================*/
/* Index: account_name                                          */
/*==============================================================*/
create unique index if not exists  account_name on t_pay_account (
name
);

/*==============================================================*/
/* Table: t_payment                                             */
/*==============================================================*/
create table if not exists  t_payment (
   id                   SERIAL not null,
   batch                VARCHAR              null,
   policy_no            VARCHAR              null,
   transfer_no          VARCHAR              null,
   transfer_amount      FLOAT8               null,
   creator              INT8                 null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   updated_by           INT8                 null,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   domain_id            INT8                 null,
   addi                 JSONB                null,
   remark               VARCHAR              null,
   status               VARCHAR              null,
   constraint PK_T_PAYMENT primary key (id)
);

comment on table t_payment is
'缴费表(用于对公转账自动化)';

comment on column t_payment.id is
'编码';

comment on column t_payment.batch is
'批次';

comment on column t_payment.policy_no is
'保单号';

comment on column t_payment.transfer_no is
'转账流水号';

comment on column t_payment.transfer_amount is
'金额';

comment on column t_payment.creator is
'创建者用户ID';

comment on column t_payment.create_time is
'创建时间';

comment on column t_payment.updated_by is
'更新者';

comment on column t_payment.update_time is
'修改时间';

comment on column t_payment.domain_id is
'数据属主';

comment on column t_payment.addi is
'附加数据';

comment on column t_payment.remark is
'备注';

comment on column t_payment.status is
'状态, 未缴费: 0, 已缴费: 2, 作废: 4';

/*==============================================================*/
/* Table: t_practice                                            */
/*==============================================================*/
create table if not exists  t_practice (
   id                   SERIAL not null,
   paper_id             INT8                 null,
   exam_paper_id        INT8                 null,
   name                 VARCHAR(100)         null,
   correct_mode         VARCHAR(50)          null
      constraint CK_CORRECT_MODE check (correct_mode is null or (correct_mode IN ('00','02'))),
   type                 VARCHAR(50)          null
      constraint CK_TYPE check (type is null or (type IN ('00','02', '04'))),
   creator              INT8                 not null,
   create_time          INT8                 null,
   updated_by           INT8                 null,
   update_time          INT8                 null,
   addi                 JSONB                null,
   status               VARCHAR(50)          null
      constraint CK_STATUS check (status is null or (status IN ('00','02', '04'))),
   allowed_attempts     INT4                 null,
   constraint PK_T_PRACTICE primary key (id)
);

comment on table t_practice is
'练习表';

comment on column t_practice.id is
'练习ID';

comment on column t_practice.paper_id is
'练习选定的试卷ID';

comment on column t_practice.exam_paper_id is
'考卷ID';

comment on column t_practice.name is
'练习名称';

comment on column t_practice.correct_mode is
'批改方式，00：自动（AI）  10：手动';

comment on column t_practice.type is
'练习类型，00：经典  02：常练  04：智能';

comment on column t_practice.creator is
'创建者';

comment on column t_practice.create_time is
'创建时间';

comment on column t_practice.updated_by is
'更新者';

comment on column t_practice.update_time is
'更新时间';

comment on column t_practice.addi is
'附加信息';

comment on column t_practice.status is
'状态， 00：未发布  02：未发布  04：已删除';

comment on column t_practice.allowed_attempts is
'可作答的次数，0：不限制次数 大于0：相应的次数';

ALTER SEQUENCE t_practice_id_seq RESTART WITH 2000;

/*==============================================================*/
/* Table: t_practice_student                                    */
/*==============================================================*/
create table if not exists  t_practice_student (
   id                   INT4                 not null,
   practice_id          INT4                 not null,
   student_id           INT4                 not null,
   addi                 JSONB                null,
   creator              INT4                 null,
   updated_by           INT4                 null,
   create_time          INT8                 null,
   update_time          INT8                 null,
   status               VARCHAR(128)         null,
   constraint PK_T_PRACTICE_STUDENT primary key (id)
);

comment on table t_practice_student is
't_practice_student';

comment on column t_practice_student.id is
'主键';

comment on column t_practice_student.practice_id is
'practice_id';

comment on column t_practice_student.student_id is
'学生id';

comment on column t_practice_student.addi is
'addi';

comment on column t_practice_student.creator is
'creator';

comment on column t_practice_student.updated_by is
'updated_by';

comment on column t_practice_student.create_time is
'create_time';

comment on column t_practice_student.update_time is
'update_time';

comment on column t_practice_student.status is
'00：正常 02：被删除';

/*==============================================================*/
/* Table: t_practice_submissions                                */
/*==============================================================*/
create table if not exists  t_practice_submissions (
   id                   SERIAL               not null,
   exam_paper_id        INT8                 null,
   student_id           bigint               null,
   practice_id          bigint               null,
   start_time           INT8                 null,
   end_time             INT8                 null,
   last_start_time      INT8                 null,
   last_end_time        INT8                 null,
   elapsed_seconds      INT8                 null,
   attempt              INT4                 null,
   wrong_attempt        INT4                 null,
   remark               VARCHAR              null,
   creator              INT8                 not null,
   create_time          INT8                 null,
   updated_by           INT8                 null,
   update_time          INT8                 null,
   status               VARCHAR              null default '00',
   addi                 JSONB                null,
   constraint PK_T_PRACTICE_SUBMISSIONS primary key (id)
);

comment on table t_practice_submissions is
't_student_practice';

comment on column t_practice_submissions.id is
'学生练习ID';

comment on column t_practice_submissions.exam_paper_id is
'考卷ID';

comment on column t_practice_submissions.student_id is
'学生ID';

comment on column t_practice_submissions.practice_id is
'练习ID';

comment on column t_practice_submissions.start_time is
'开始答题时间';

comment on column t_practice_submissions.end_time is
'结束答题时间';

comment on column t_practice_submissions.last_start_time is
'最近一次进入作答的时间';

comment on column t_practice_submissions.last_end_time is
'最近一次退出作答页面的时间（未提交）';

comment on column t_practice_submissions.elapsed_seconds is
'这一次练习过去了的时间';

comment on column t_practice_submissions.attempt is
'当前是第几次作答这个练习';

comment on column t_practice_submissions.wrong_attempt is
'学生进入一次练习提交中错题集的次数';

comment on column t_practice_submissions.remark is
'备注';

comment on column t_practice_submissions.creator is
'创建者';

comment on column t_practice_submissions.create_time is
'创建时间';

comment on column t_practice_submissions.updated_by is
'更新者';

comment on column t_practice_submissions.update_time is
'更新时间';

comment on column t_practice_submissions.status is
'状态 00：允许作答 02 ：不允许作答 04：删除  06：已提交 08：已批改';

comment on column t_practice_submissions.addi is
'附加信息';

/*==============================================================*/
/* Table: t_price                                               */
/*==============================================================*/
create table if not exists  t_price (
   id                   SERIAL not null,
   title                VARCHAR              null,
   category             VARCHAR              null,
   insurance_type_id    INT8                 null,
   org_name             VARCHAR              null,
   province             VARCHAR              null,
   city                 VARCHAR              null,
   district             VARCHAR              null,
   guaranteed_projects  JSONB                null,
   extra_projects       JSONB                null,
   price_config         JSONB                null,
   files                JSONB                null,
   is_default           BOOL                 null,
   creator_id           INT8                 null,
   create_time          INT8                 null,
   updated_by           INT8                 null,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   domain_id            INT8                 null,
   addi                 JSONB                null,
   remark               VARCHAR              null,
   status               VARCHAR              null,
   constraint PK_T_PRICE primary key (id)
);

comment on table t_price is
'价格设置表
';

comment on column t_price.id is
'价格id';

comment on column t_price.title is
'标题';

comment on column t_price.category is
'类型';

comment on column t_price.insurance_type_id is
'险种id';

comment on column t_price.org_name is
'投保单位';

comment on column t_price.province is
'省';

comment on column t_price.city is
'市';

comment on column t_price.district is
'区/县';

comment on column t_price.guaranteed_projects is
'保障项目';

comment on column t_price.extra_projects is
'附加条款';

comment on column t_price.price_config is
'价格配置';

comment on column t_price.files is
'模板路径 ';

comment on column t_price.is_default is
'是否默认值';

comment on column t_price.creator_id is
'创建者id';

comment on column t_price.create_time is
'创建时间';

comment on column t_price.updated_by is
'更新者';

comment on column t_price.update_time is
'修改时间';

comment on column t_price.domain_id is
'数据属主';

comment on column t_price.addi is
'备用字段';

comment on column t_price.remark is
'备注';

comment on column t_price.status is
'状态，0：有效，2：无效';

ALTER SEQUENCE t_price_id_seq RESTART WITH 20000;
INSERT INTO t_price(id, insurance_type_id, org_name,province,city,district,price_config,is_default, status)
 VALUES (1000,10000, '','','','','{
        "others":{
            "SpacingToNextLevel":10
        },
        "比赛类":{
            "day_0to29_reqmin":500, "day_0to29_reqmax":5000,
            "day_0to29_student_teacher_price":500,
            "day_0to29_adult_price":500, "day_0to29_adult_m":1,
            "day_EQ_30_price":3000, "day_EQ_30_reqmin":500, "day_EQ_30_reqmax":5000,
            "day_31to61_price":4000, "day_31to61_reqmin":2000, "day_31to61_reqmax":6000,
            "day_62to92_price":5000, "day_62to92_reqmin":3000, "day_62to92_reqmax":8000,
            "day_93to182_price":8000, "day_93to182_reqmin":4000, "day_93to182_reqmax":10000,
            "day_183to364_price":12000, "day_183to364_reqmin":8000, "day_183to364_reqmax":20000
        },
        "活动类":{
            "day_0to10_count_LTE_50_price":500,
            "day_11to19_count_LTE_50_price":2000,
            "day_20to29_count_LTE_50_price":3000,
            "day_1_count_GT_50_price":400,
            "day_2_count_GT_50_price":500,
            "day_3to10_count_GT_50_price":200,
            "day_11to19_count_GT_50_price":2000,
            "day_20to29_count_GT_50_price":3000,
            "day_EQ_30_price": 3000
        },
        "monthly_fee": {
        "level1": 3000,
        "level2": 4000,
        "level3": 5000,
        "level4": 8000,
        "level5": 12000
    }
       }', true,0);


insert into t_price(id, insurance_type_id,org_name,province,city,district, guaranteed_projects, price_config, is_default, status)
values(1002, 10022,'','','','','[{"sn": 1, "title": "每所学校(幼儿园)每次事故赔偿限额", "content": "人民币800万元"},
       {"sn": 2, "title": "每所学校每年累计赔偿限额", "content": "人民币2000万元"},
       {"sn": 3, "title": "每所学校每次事故财产损失赔偿限额", "content": "人民币200万元"},
       {"sn": 4, "title": "每人每年赔偿限额", "content": "人民币60万元"},
       {"sn": 5, "title": "每所学校每次事故法律费用赔偿限额", "content": "人民币20万元"},
       {"sn": 6, "title": "每次事故每人财产损失赔偿限额", "content": "人民币5万元"}]',
       '{
        "compulsoryEdu": {
            "main": 500,
            "unit": "元/人/年",
            "noFault": 200
        },
        "nonCompulsoryEdu": {
            "main": 700,
            "unit": "元/人/年",
            "noFault": 200
        }
        }',
       true, '0' );



insert into t_price(id, insurance_type_id,org_name,province,city,district, guaranteed_projects, price_config, is_default, status)
values(1004, 10024, '','','','','{
    "addi": [
        {
            "list": [
                {
                    "sn": 1,
                    "title": "意外伤害医疗",
                    "content": "3万"
                },
                {
                    "sn": 2,
                    "title": "意外死亡或伤残",
                    "content": "30万"
                }
            ],
            "name": "附加24小时意外保险"
        }
    ],
    "scheme": [
        {
            "list": [
                {
                    "sn": 1,
                    "title": "身故赔偿",
                    "content": "50万"
                },
                {
                    "sn": 2,
                    "title": "伤残赔偿",
                    "content": "80万"
                },
                {
                    "sn": 3,
                    "title": "额外费用赔偿",
                    "content": "4万"
                },
                {
                    "sn": 4,
                    "title": "医疗费用赔偿",
                    "content": "5万"
                },
                {
                    "sn": 5,
                    "title": "诉讼费用赔偿",
                    "content": "5万"
                }
            ],
            "name": "方案一"
        },
        {
            "list": [
                {
                    "sn": 1,
                    "title": "身故赔偿",
                    "content": "60万"
                },
                {
                    "sn": 2,
                    "title": "伤残赔偿",
                    "content": "100万"
                },
                {
                    "sn": 3,
                    "title": "额外费用赔偿",
                    "content": "4万"
                },
                {
                    "sn": 4,
                    "title": "医疗费用赔偿",
                    "content": "6万"
                },
                {
                    "sn": 5,
                    "title": "诉讼费用赔偿",
                    "content": "5万"
                }
            ],
            "name": "方案二"
        }
    ]
}','{
    "addi": [
        {
            "sn": 3,
            "name": "附加24小时意外保险",
            "unit": "元/人/年",
            "price": 5000,
            "tick_axis": "C26",
            "amount_axis": "E26"
        }
    ],
    "unit": "元/人/年",
    "scheme": [
        {
            "sn": 1,
            "name": "方案一",
            "unit": "元/人/年",
            "price": 5000,
            "tick_axis": "C24",
            "amount_axis": "E24"
        },
        {
            "sn": 2,
            "name": "方案二",
            "unit": "元/人/年",
            "price": 8000,
            "tick_axis": "C25",
            "amount_axis": "E25"
        }
    ]
}',true, '0' );



insert into t_price(id, insurance_type_id,org_name,province,city,district, guaranteed_projects, price_config, is_default, status)
values(1006, 10026, '','','','','[
    {
        "sn": 1,
        "title": "每生每年医疗费用赔偿限额",
        "content": "15万元"
    },
    {
        "sn": 2,
        "title": "每生每年赔偿限额",
        "content": "60万元"
    },
    {
        "sn": 3,
        "title": "每所学校每次事故赔偿限额",
        "content": "2500万元"
    },
    {
        "sn": 4,
        "title": "每所学校每年累计赔偿限额",
        "content": "5000万元"
    },
    {
        "sn": 5,
        "title": "每次事故法律费用限额",
        "content": "20万元"
    },
    {
        "sn": 6,
        "title": "每校每年法律费用限额",
        "content": "200万元"
    },
    {
        "sn": 7,
        "title": "每生每次附加第三者责任限额",
        "content": "20万元"
    },
    {
        "sn": 8,
        "title": "每校每年附加第三者责任限额",
        "content": "200万元"
    },
    {
        "sn": 9,
        "title": "每生每次事故附加精神损害费",
        "content": "12万元"
    },
    {
        "sn": 10,
        "title": "每校每年附加精神损害费限额",
        "content": "120万元"
    },
    {
        "sn": 11,
        "title": "扩展实习无过失道义性补偿限额(含医疗费用)",
        "content": "10万元"
    }
]','{
    "axis": "B26",
    "unit": "元/人/年",
    "price": 5000
}',true, '0' );



insert into t_price(id, insurance_type_id, org_name,province,city,district,guaranteed_projects, price_config, is_default, status)
values(1008, 10028,'','','','', '[
    {
        "sn": 1,
        "title": "每个座位赔付限额",
        "content": "30万（每个座位意外医疗费用赔付限额5万）"
    },
    {
        "sn": 2,
        "title": "每台车辆累计赔付限额",
        "content": "投保座位数*30万元"
    }
]','{
    "unit": "元/座/年",
    "driver": 10000,
    "general": 5000,
    "StartRow": 17,
    "BankNoCol": 2,
    "SeatNumCol": 4,
    "RoadLevelCol": 5,
    "LicensePlateCol": 1
}',true, '0' );




insert into t_price(id, insurance_type_id, org_name,province,city,district,guaranteed_projects, price_config, is_default, status)
values(1010, 10030,'','','','', '[
    {
        "sn": 1,
        "title": "累计赔偿限额",
        "content": "600万元"
    },
    {
        "sn": 2,
        "title": "每次事故赔偿限额",
        "content": "80万元"
    },
    {
        "sn": 3,
        "title": "每人每次事故赔偿限额",
        "content": "8万元"
    }
]','{
    "unit": "元",
    "price": [
        {
            "max": 500,
            "min": 0,
            "canteen": 70000,
            "tuckShop": 35000,
            "TickShopAxis": "D18",
            "TickCanteenAxis": "F18"
        },
        {
            "max": 1000,
            "min": 500,
            "canteen": 140000,
            "tuckShop": 70000,
            "TickShopAxis": "D19",
            "TickCanteenAxis": "F19"
        },
        {
            "max": 1500,
            "min": 1000,
            "canteen": 170000,
            "tuckShop": 85000,
            "TickShopAxis": "D20",
            "TickCanteenAxis": "F20"
        },
        {
            "max": 2000,
            "min": 1500,
            "canteen": 200000,
            "tuckShop": 100000,
            "TickShopAxis": "D21",
            "TickCanteenAxis": "F21"
        },
        {
            "max": 2500,
            "min": 2000,
            "canteen": 240000,
            "tuckShop": 120000,
            "TickShopAxis": "D22",
            "TickCanteenAxis": "F22"
        },
        {
            "max": 3000,
            "min": 2500,
            "canteen": 300000,
            "tuckShop": 150000,
            "TickShopAxis": "D23",
            "TickCanteenAxis": "F23"
        },
        {
            "max": 4000,
            "min": 3000,
            "canteen": 400000,
            "tuckShop": 200000,
            "TickShopAxis": "D24",
            "TickCanteenAxis": "F24"
        },
        {
            "max": 100000,
            "min": 4000,
            "canteen": 500000,
            "tuckShop": 250000,
            "TickShopAxis": "D25",
            "TickCanteenAxis": "F25"
        }
    ]
}',true, '0' );

--------- 比赛组织方
insert into t_price(id, insurance_type_id, title, price_config, is_default,status)
values (
    1020,10060,'累计赔偿限额300万(每人人身伤害赔偿限额30万)','{
        "DefaultDays": 1,
        "Standard":300000,
        "IncreasePerDay":30000,
        "DecreasePerDay":100000,
        "NegotiatedNeedDays":1,
        "MandatoryNegotiation":false,
        "SuddenDeathCompensation":100000,
        "IndemnityLimit":300000000,
        "Tariff":0.1
    }',true,'0'
);

insert into t_price(id, insurance_type_id, title, price_config, is_default, status)
values (
    1022,10060,'累计赔偿限额500万(每人人身伤害赔偿限额50万)','{
        "DefaultDays": 1,
        "Standard":500000,
        "IncreasePerDay":50000,
        "DecreasePerDay":100000,
        "NegotiatedNeedDays":1,
        "MandatoryNegotiation":false,
        "SuddenDeathCompensation":100000,
        "IndemnityLimit":500000000,
        "Tariff":0.1
    }',true,'0'
);

insert into t_price(id, insurance_type_id, title, price_config, is_default, status)
values (
    1024,10060,'累计赔偿限额1000万(每人人身伤害赔偿限额50万)','{
        "DefaultDays": 1,
        "Standard":1000000,
        "IncreasePerDay":50000,
        "DecreasePerDay":100000,
        "NegotiatedNeedDays": 1,        
        "MandatoryNegotiation":true,
        "SuddenDeathCompensation":100000,
        "IndemnityLimit":1000000000,
        "Tariff":0.1
    }',true,'0'
);
-- ------------ 俱乐部场地
insert into t_price(id, insurance_type_id, category, title, price_config, is_default, status)
values (
    1030,10070,'俱乐部','累计赔偿限额300万(每人人身伤害赔偿限额30万)','{
        "DefaultNum": 1,
        "Standard":300000,
        "IncreasePerGround":200000,
        "OpeningMultiple":2,
        "NegotiatedNeedNum":2,
        "SuddenDeathCompensation":100000,
        "LimitPerGround":500,
        "IncreaseOverLimit":50000,
        "IndemnityLimit":300000000,
        "Tariff":0.1
    }',true,'0'
);

insert into t_price(id, insurance_type_id, category, title, price_config, is_default, status)
values (
    1032,10070,'俱乐部','累计赔偿限额500万(每人人身伤害赔偿限额50万)','{
        "DefaultNum": 1,
        "Standard":500000,
        "IncreasePerGround":200000,
        "OpeningMultiple":2,
        "NegotiatedNeedNum":2,
        "SuddenDeathCompensation":100000,
        "LimitPerGround":500,
        "IncreaseOverLimit":50000,
        "IndemnityLimit":500000000,
        "Tariff":0.1
    }',true,'0'
);

insert into t_price(id, insurance_type_id, category, title, price_config, is_default, status)
values (
    1034,10070,'场地','累计赔偿限额300万(每人人身伤害赔偿限额30万)','{
        "DefaultNum": 1,
        "Standard":300000,
        "IncreasePerGround":200000,
        "OpeningMultiple":2,
        "NegotiatedNeedNum":2,
        "SuddenDeathCompensation":100000,
        "IndemnityLimit":300000000,
        "Tariff":0.1
    }',true,'0'
);

insert into t_price(id, insurance_type_id, category, title, price_config, is_default, status)
values (
    1036,10070,'场地','累计赔偿限额500万(每人人身伤害赔偿限额50万)','{
        "DefaultNum": 1,
        "Standard":500000,
        "IncreasePerGround":200000,
        "OpeningMultiple":2,
        "NegotiatedNeedNum":2,
        "SuddenDeathCompensation":100000,
        "IndemnityLimit":500000000,
        "Tariff":0.1
    }',true,'0'
);

------- 游泳池

insert into t_price(id, insurance_type_id, price_config, is_default, status)
values (
    1040,10080,'{
        "OpenPoolUnit": 1000000,
        "TrainingPoolUnit":500000,
        "OpenHeatedPoolUnit":500000,
        "UnOpenHeatedPoolUnit":200000,
        "NegotiatedNeedNum":2,
        "IndemnityLimit":500000000,
        "Tariff":0.1
    }',true,'0'
);

update t_price set province='' where province is null;
update t_price set city='' where city is null;
update t_price set district='' where district is null;
update t_price set org_name='' where org_name is null;



/*==============================================================*/
/* Table: t_prj                                                 */
/*==============================================================*/
create table if not exists  t_prj (
   id                   SERIAL not null,
   name                 VARCHAR              null,
   limn                 VARCHAR              null,
   price                NUMERIC              null,
   cycle                integer              null,
   issuer               INT8                 null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   issue_time           INT8                 null,
   deadline             INT8                 null,
   remark               VARCHAR              null,
   status               VARCHAR              null,
   constraint PK_T_PRJ primary key (id)
);

comment on table t_prj is
'项目信息表';

comment on column t_prj.id is
'编号';

comment on column t_prj.name is
'项目名称';

comment on column t_prj.limn is
'项目描述';

comment on column t_prj.price is
'报价';

comment on column t_prj.cycle is
'期望周期，以自然日为单位';

comment on column t_prj.issuer is
'发布者编号，四方';

comment on column t_prj.create_time is
'创建时间';

comment on column t_prj.issue_time is
'发布时间';

comment on column t_prj.deadline is
'截止时间';

comment on column t_prj.remark is
'备注';

comment on column t_prj.status is
'draft,未发布
isuued,已发布
cancelled,取消
engaged,确定了承接人
signed,签订合同
finished,项目完成';

ALTER SEQUENCE t_prj_id_seq RESTART WITH 10000;

/*==============================================================*/
/* Table: t_proof                                               */
/*==============================================================*/
create table if not exists  t_proof (
   id                   SERIAL not null,
   user_id              INT8                 null,
   expertise_id         INT8                 null,
   limn                 VARCHAR              null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   constraint PK_T_PROOF primary key (id)
);

comment on table t_proof is
'人才知识能力领域说明表';

comment on column t_proof.id is
'编号';

comment on column t_proof.user_id is
'用户编号';

comment on column t_proof.expertise_id is
'知识能力领域编号';

comment on column t_proof.limn is
'能力描述';

comment on column t_proof.create_time is
'创建时间';

comment on column t_proof.update_time is
'更新时间';

ALTER SEQUENCE t_proof_id_seq RESTART WITH 10000;

/*==============================================================*/
/* Table: t_prove                                               */
/*==============================================================*/
create table if not exists  t_prove (
   id                   SERIAL not null,
   proof_id             INT8                 null,
   judgement            VARCHAR              null,
   creator              INT8                 null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   constraint PK_T_PROVE primary key (id)
);

comment on table t_prove is
'知识能力领域鉴定、证明表';

comment on column t_prove.id is
'编号';

comment on column t_prove.proof_id is
'被鉴定材料编号';

comment on column t_prove.judgement is
'鉴定结论';

comment on column t_prove.creator is
'鉴定者';

comment on column t_prove.create_time is
'鉴定时间';

comment on column t_prove.update_time is
'鉴定更新时间';

/*==============================================================*/
/* Table: t_qualification                                       */
/*==============================================================*/
create table if not exists  t_qualification (
   id                   SERIAL not null,
   user_id              INT8                 null,
   expertise_id         INT8                 null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   constraint PK_T_QUALIFICATION primary key (id)
);

comment on table t_qualification is
'人才资质表';

comment on column t_qualification.id is
'资质证明编号';

comment on column t_qualification.user_id is
'用户编号';

comment on column t_qualification.expertise_id is
'专长编号';

comment on column t_qualification.create_time is
'创建时间';

/*==============================================================*/
/* Table: t_question                                            */
/*==============================================================*/
create table if not exists  t_question (
   id                   SERIAL               not null,
   type                 VARCHAR(64)          not null,
   content              TEXT                 null,
   options              JSONB                null,
   answers              JSONB                null,
   score                FLOAT8               null,
   difficulty           INT4                 not null,
   tags                 JSONB                null,
   analysis             TEXT                 null,
   title                TEXT                 null,
   answer_file_path     JSONB                null,
   test_file_path       JSONB                null,
   input                VARCHAR(128)         null,
   output               VARCHAR(128)         null,
   example              JSONB                null,
   repo                 JSONB                null,
   "order"              INT8                 null,
   creator              INT8                 not null,
   create_time          INT8                 not null,
   updated_by           INT8                 not null,
   update_time          INT8                 not null,
   addi                 JSONB                null,
   status               VARCHAR(64)          not null default '00',
   question_attachments_path JSONB                null,
   access_mode          VARCHAR(4)           not null default '00',
   belong_to            INT8                 null,
   constraint PK_T_QUESTION primary key (id)
);

comment on table t_question is
'题目表';

comment on column t_question.id is
'编号';

comment on column t_question.type is
'类型  00:单选题  02:多选题 04:判断题 06:填空题 08:简答题 10:编程题';

comment on column t_question.content is
'题目内容';

comment on column t_question.options is
'题目选项';

comment on column t_question.answers is
'题目答案';

comment on column t_question.score is
'题目分值';

comment on column t_question.difficulty is
'题目难度 1:简单  2:中等 3:困难';

comment on column t_question.tags is
'题目标签';

comment on column t_question.analysis is
'题目解析';

comment on column t_question.title is
'编程题目题干';

comment on column t_question.answer_file_path is
'答案文件路径';

comment on column t_question.test_file_path is
'测试文件路径';

comment on column t_question.input is
'输入';

comment on column t_question.output is
'输出';

comment on column t_question.example is
'示例';

comment on column t_question.repo is
'仓库';

comment on column t_question."order" is
'顺序';

comment on column t_question.creator is
'创建者';

comment on column t_question.create_time is
'创建时间';

comment on column t_question.updated_by is
'更新者';

comment on column t_question.update_time is
'更新时间';

comment on column t_question.addi is
'附加信息';

comment on column t_question.status is
'状态，00:正常 02:作废 04:异常';

comment on column t_question.question_attachments_path is
'题目附件url数组';

comment on column t_question.access_mode is
'题目访问权限，00私有 02共享 04公开';

comment on column t_question.belong_to is
'归属于某个题库';

/*==============================================================*/
/* Index: idx_question_id                                       */
/*==============================================================*/
create unique index if not exists  idx_question_id on t_question (
id
);

/*==============================================================*/
/* Index: idx_question_id_creator                               */
/*==============================================================*/
create unique index if not exists  idx_question_id_creator on t_question (
( id ),
( creator )
);

/*==============================================================*/
/* Table: t_question_bank                                       */
/*==============================================================*/
create table if not exists  t_question_bank (
   id                   SERIAL               not null,
   domain_id            INT8                 not null,
   type                 VARCHAR(64)          not null,
   name                 VARCHAR(64)          not null default '未命名题库',
   tags                 JSONB                null,
   repos                JSONB                null,
   default_repo         VARCHAR(64)          null,
   creator              INT8                 not null,
   create_time          INT8                 not null,
   updated_by           INT8                 not null,
   update_time          INT8                 not null,
   remark               VARCHAR(128)         null,
   addi                 JSONB                null,
   status               VARCHAR(64)          not null default '00',
   constraint PK_T_QUESTION_BANK primary key (id)
);

comment on table t_question_bank is
'题库表';

comment on column t_question_bank.id is
'编号';

comment on column t_question_bank.domain_id is
'所属域ID';

comment on column t_question_bank.type is
'类型： 00:理论题库，02:编程题库';

comment on column t_question_bank.name is
'名字';

comment on column t_question_bank.tags is
'标签';

comment on column t_question_bank.repos is
'仓库';

comment on column t_question_bank.default_repo is
'题库git repo';

comment on column t_question_bank.creator is
'创建者';

comment on column t_question_bank.create_time is
'创建时间';

comment on column t_question_bank.updated_by is
'更新者';

comment on column t_question_bank.update_time is
'更新时间';

comment on column t_question_bank.remark is
'备注';

comment on column t_question_bank.addi is
'附加信息';

comment on column t_question_bank.status is
'状态，00:正常 02:作废 04:异常';

/*==============================================================*/
/* Index: idx_question_bank_id                                  */
/*==============================================================*/
create unique index if not exists  idx_question_bank_id on t_question_bank (
id
);

/*==============================================================*/
/* Index: idx_question_bank_id_creator                          */
/*==============================================================*/
create unique index if not exists  idx_question_bank_id_creator on t_question_bank (
id,
creator
);

/*==============================================================*/
/* Table: t_region                                              */
/*==============================================================*/
create table if not exists  t_region (
   id                   SERIAL not null,
   region_name          VARCHAR              null,
   code                 INT8                 null,
   region_short_name    VARCHAR              null,
   parent_id            INT8                 null,
   level                INT8                 null,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   creator              INT8                 null,
   domain_id            INT8                 null,
   addi                 JSONB                null,
   remark               VARCHAR              null,
   status               VARCHAR              null,
   constraint PK_T_REGION primary key (id)
);

comment on table t_region is
'区域列表';

comment on column t_region.id is
'区域编号';

comment on column t_region.region_name is
'区域名称';

comment on column t_region.code is
'区域行政编码';

comment on column t_region.region_short_name is
'区域缩写';

comment on column t_region.parent_id is
'区域的父级id, 定义: 省级没有父级, parent_id为0; 市级的父级是省; 区县的父级是市';

comment on column t_region.level is
'地区级别: 2-省,4-市,6-区/县';

comment on column t_region.update_time is
'可能以后存在着一年更新一次区域表';

comment on column t_region.creator is
'本数据创建者';

comment on column t_region.domain_id is
'数据隶属';

comment on column t_region.addi is
'附加信息';

comment on column t_region.remark is
'备注';

comment on column t_region.status is
'0:有效, 2: 删除, 过了一段时间有些区域可能会被删除';

/*==============================================================*/
/* Index: Idx_region_id_pid                                     */
/*==============================================================*/
create  index if not exists  Idx_region_id_pid on t_region (
id,
parent_id
);

/*==============================================================*/
/* Index: idx_region_name                                       */
/*==============================================================*/
create  index if not exists  idx_region_name on t_region (
region_name
);

/*==============================================================*/
/* Table: t_relation                                            */
/*==============================================================*/
create table if not exists  t_relation (
   id                   SERIAL not null,
   left_id              INT8                 null,
   left_type            VARCHAR              null,
   left_key             VARCHAR              null,
   left_key_type        VARCHAR              null,
   kind                 VARCHAR              not null,
   right_id             INT8                 null,
   right_type           VARCHAR              null,
   right_key            VARCHAR              null,
   right_value_type     VARCHAR              null,
   right_value          VARCHAR              null,
   rule_area            VARCHAR              null,
   creator              INT8                 null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   updated_by           INT8                 null,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   domain_id            INT8                 null,
   addi                 JSONB                null,
   remark               VARCHAR              null,
   status               VARCHAR              null,
   constraint PK_T_RELATION primary key (id)
);

comment on table t_relation is
'描述两个实体间的隶属关系，类似于master:detail，校快保，描述销售/学校管理员/学校统计员与学校间的对应关系



left_key_type -- 左识别标识类型，帐号: account, 邮箱: email, 手机: tel, 微信公众号openID: mp_open_id, 微信开放平台openID: wx_open_id
+{left_id,left_key} --左键(表中的主键,如果主键类型是int，则为left_id, 否则是left_key)
+kind --关系类型, 例如, 管理员与学校, 组员与组
+right_key_type -- 意义与左识别标识类型相同
+{right_id,right_key} --意义与左键相同

例如
left_type          left_id kind         right_type     right_id
''t_user.id'',       1000,   ''学校:管理员'', ''t_school.id'', 2273

left_type          left_key kind         right_type     right_key
''t_user.account'',  ''ax992'', ''保安:门岗'',  ''t_gate.name'', ''南门''  ';

comment on column t_relation.id is
'编号';

comment on column t_relation.left_id is
'左编号';

comment on column t_relation.left_type is
'左类型，用户编号: t_user.id';

comment on column t_relation.left_key is
'左识别标识';

comment on column t_relation.left_key_type is
'左识别标识类型，帐号: account, 邮箱: email, 手机: tel, 微信公众号openID: mp_open_id, 微信开放平台openID: wx_open_id';

comment on column t_relation.kind is
'关系类型';

comment on column t_relation.right_id is
'目标资源编号，如学校ID';

comment on column t_relation.right_type is
'右类型，如t_school.id';

comment on column t_relation.right_key is
'右识别模块';

comment on column t_relation.right_value_type is
'右值数据类型，默认为int8,则值存储于right_id,其它类型则存储于right_value中';

comment on column t_relation.right_value is
'右值(非int8类型)';

comment on column t_relation.rule_area is
'管辖地区';

comment on column t_relation.creator is
'创建者用户ID';

comment on column t_relation.create_time is
'创建时间';

comment on column t_relation.updated_by is
'更新者';

comment on column t_relation.update_time is
'修改时间';

comment on column t_relation.domain_id is
'数据属主';

comment on column t_relation.addi is
'附加数据';

comment on column t_relation.remark is
'备注';

comment on column t_relation.status is
'状态';

ALTER SEQUENCE t_relation_id_seq RESTART WITH 20000;

-- drop table if exists t_relation_history;
-- create table if not exists t_relation_history as select * from t_relation;

/*==============================================================*/
/* Index: idx_id_id_relation                                    */
/*==============================================================*/
create unique index if not exists  idx_id_id_relation on t_relation (
left_id,
left_type,
kind,
right_id,
right_type
);

/*==============================================================*/
/* Index: idx_key_id_relation                                   */
/*==============================================================*/
create unique index if not exists  idx_key_id_relation on t_relation (
left_type,
left_key,
kind,
right_id,
right_type
);

/*==============================================================*/
/* Index: idx_id_key_relation                                   */
/*==============================================================*/
create unique index if not exists  idx_id_key_relation on t_relation (
left_id,
left_type,
kind,
right_type,
right_key
);

/*==============================================================*/
/* Index: idx_key_key_relation                                  */
/*==============================================================*/
create unique index if not exists  idx_key_key_relation on t_relation (
left_type,
left_key,
kind,
right_type,
right_key
);

/*==============================================================*/
/* Table: t_relation_history                                    */
/*==============================================================*/
create table if not exists  t_relation_history (
   id                   INT8                 not null,
   left_id              INT8                 null,
   left_type            VARCHAR              null,
   left_key             VARCHAR              null,
   left_key_type        VARCHAR              null,
   kind                 VARCHAR              not null,
   right_id             INT8                 null,
   right_type           VARCHAR              null,
   right_key            VARCHAR              null,
   right_value_type     VARCHAR              null,
   right_value          VARCHAR              null,
   rule_area            VARCHAR              null,
   creator              INT8                 null,
   create_time          INT8                 null,
   updated_by           INT8                 null,
   update_time          INT8                 null,
   domain_id            INT8                 null,
   addi                 JSONB                null,
   remark               VARCHAR              null,
   status               VARCHAR              null,
   sn                   SERIAL not null,
   constraint PK_T_RELATION_HISTORY primary key (sn)
);

comment on table t_relation_history is
'关系变更历史';

comment on column t_relation_history.id is
'编号';

comment on column t_relation_history.left_id is
'左编号';

comment on column t_relation_history.left_type is
'左类型，用户编号: t_user.id';

comment on column t_relation_history.left_key is
'左识别标识';

comment on column t_relation_history.left_key_type is
'左识别标识类型，帐号: account, 邮箱: email, 手机: tel, 微信公众号openID: mp_open_id, 微信开放平台openID: wx_open_id';

comment on column t_relation_history.kind is
'关系类型';

comment on column t_relation_history.right_id is
'目标资源编号，如学校ID';

comment on column t_relation_history.right_type is
'右类型，如t_school.id';

comment on column t_relation_history.right_key is
'右识别模块';

comment on column t_relation_history.right_value_type is
'右值数据类型，默认为int8,则值存储于right_id,其它类型则存储于right_value中';

comment on column t_relation_history.right_value is
'右值(非int8类型)';

comment on column t_relation_history.rule_area is
'管辖地区';

comment on column t_relation_history.creator is
'创建者用户ID';

comment on column t_relation_history.create_time is
'创建时间';

comment on column t_relation_history.updated_by is
'更新者';

comment on column t_relation_history.update_time is
'修改时间';

comment on column t_relation_history.domain_id is
'数据属主';

comment on column t_relation_history.addi is
'附加数据';

comment on column t_relation_history.remark is
'备注';

comment on column t_relation_history.status is
'状态';

comment on column t_relation_history.sn is
'primary key';

/*==============================================================*/
/* Index: idx_relation_history                                  */
/*==============================================================*/
create  index if not exists  idx_relation_history on t_relation_history (
left_id,
left_key,
kind,
right_id
);

/*==============================================================*/
/* Table: t_report_claims                                       */
/*==============================================================*/
create table if not exists  t_report_claims (
   id                   SERIAL not null,
   informant_id         INT8                 null,
   informant            JSONB                null,
   insured_id           INT8                 null,
   insured              JSONB                null,
   insurance_type       VARCHAR              null,
   insurance_type_id    INT8                 null,
   insurance_policy_sn  VARCHAR              null,
   insurance_policy_id  INT8                 null,
   insurance_policy_start INT8                 null,
   insurance_policy_cease INT8                 null,
   report_sn            VARCHAR              null,
   insured_channel      VARCHAR              not null,
   insured_org          VARCHAR              null,
   treatment            VARCHAR              null,
   hospital             VARCHAR              null,
   injured_location     VARCHAR              null,
   injured_part         VARCHAR              null,
   reason               VARCHAR              null,
   injured_desc         VARCHAR              null,
   credit_code          VARCHAR              null,
   bank_account_type    VARCHAR              null,
   bank_account_name    VARCHAR              null,
   bank_name            VARCHAR              null,
   bank_account_id      VARCHAR              null,
   bank_card_pic        JSONB                null,
   injured_id_pic       JSONB                null,
   guardian_id_pic      JSONB                null,
   org_lic_pic          JSONB                null,
   relation_prove_pic   JSONB                null,
   bills_pic            JSONB                null,
   hospitalized_bills_pic JSONB                null,
   invoice_pic          JSONB                null,
   medical_record_pic   JSONB                null,
   dignostic_inspection_pic JSONB                null,
   discharge_abstract_pic JSONB                null,
   other_pic            JSONB                null,
   courier_sn_pic       JSONB                null,
   paid_notice_pic      JSONB                null,
   claim_apply_pic      JSONB                null,
   equity_transfer_file JSONB                null,
   match_programme_pic  JSONB                null,
   policy_file          JSONB                null,
   addi_pic             JSONB                null,
   courier_SN           VARCHAR              null,
   reply_addr           VARCHAR              null,
   injured_time         INT8                 null,
   report_time          INT8                 not null,
   reply_time           INT8                 null,
   claims_mat_add_time  INT8                 null,
   mat_return_date      INT8                 null,
   close_date           INT8                 null,
   face_amount          FLOAT8               null,
   medi_assure_amount   FLOAT8               null,
   third_pay_amount     FLOAT8               null,
   claim_amount         FLOAT8               null,
   occurr_reason        VARCHAR              null,
   treatment_result     VARCHAR              null,
   disease_diagnosis_pic JSONB                null,
   disability_certificate JSONB                null,
   death_certificate    JSONB                null,
   student_status_certificate JSONB                null,
   refuse_desc          VARCHAR              null,
   domain_id            INT8                 null,
   creator              INT8                 null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   updated_by           INT8                 null,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   addi                 JSONB                null,
   remark               VARCHAR              null,
   status               VARCHAR              null,
   constraint PK_T_REPORT_CLAIMS primary key (id)
);

comment on table t_report_claims is
'报案理赔';

comment on column t_report_claims.id is
'编号';

comment on column t_report_claims.informant_id is
'报案人编号';

comment on column t_report_claims.informant is
'报案人';

comment on column t_report_claims.insured_id is
'被保险人编号';

comment on column t_report_claims.insured is
'被保险人';

comment on column t_report_claims.insurance_type is
'保险类型';

comment on column t_report_claims.insurance_type_id is
'险种类别';

comment on column t_report_claims.insurance_policy_sn is
'保单号';

comment on column t_report_claims.insurance_policy_id is
'保单编号 ';

comment on column t_report_claims.insurance_policy_start is
'起保时间';

comment on column t_report_claims.insurance_policy_cease is
'脱保时间';

comment on column t_report_claims.report_sn is
'报案号';

comment on column t_report_claims.insured_channel is
'投保渠道[mp,web]';

comment on column t_report_claims.insured_org is
'投保机构';

comment on column t_report_claims.treatment is
'治疗方式';

comment on column t_report_claims.hospital is
'就诊医院';

comment on column t_report_claims.injured_location is
'受伤地点';

comment on column t_report_claims.injured_part is
'受伤部位';

comment on column t_report_claims.reason is
'受伤原因';

comment on column t_report_claims.injured_desc is
'受伤过程描述';

comment on column t_report_claims.credit_code is
'统一社会信用代码';

comment on column t_report_claims.bank_account_type is
'银行账户类型';

comment on column t_report_claims.bank_account_name is
'银行账户名';

comment on column t_report_claims.bank_name is
'开户行';

comment on column t_report_claims.bank_account_id is
'银行卡号/账号';

comment on column t_report_claims.bank_card_pic is
'银行卡/存折照片';

comment on column t_report_claims.injured_id_pic is
'被保险人身份证照片';

comment on column t_report_claims.guardian_id_pic is
'监护人身份证照片';

comment on column t_report_claims.org_lic_pic is
'营业执照照片';

comment on column t_report_claims.relation_prove_pic is
'与被保险人关系证明照片';

comment on column t_report_claims.bills_pic is
'门诊费用清单照片';

comment on column t_report_claims.hospitalized_bills_pic is
'住院费用清单照片';

comment on column t_report_claims.invoice_pic is
'医疗费用发票照片';

comment on column t_report_claims.medical_record_pic is
'病历照片';

comment on column t_report_claims.dignostic_inspection_pic is
'检验检查报告照片';

comment on column t_report_claims.discharge_abstract_pic is
'出院小结照片';

comment on column t_report_claims.other_pic is
'其它资料照片';

comment on column t_report_claims.courier_sn_pic is
'快递单号照片';

comment on column t_report_claims.paid_notice_pic is
'保险金给付通知书';

comment on column t_report_claims.claim_apply_pic is
'索赔申请书';

comment on column t_report_claims.equity_transfer_file is
'权益转让书
';

comment on column t_report_claims.match_programme_pic is
'已有投保单位盖章的比赛秩序册';

comment on column t_report_claims.policy_file is
'保单文件';

comment on column t_report_claims.addi_pic is
'补充资料照片';

comment on column t_report_claims.courier_SN is
'快递单号';

comment on column t_report_claims.reply_addr is
'资料回寄地址';

comment on column t_report_claims.injured_time is
'受伤时间';

comment on column t_report_claims.report_time is
'报案时间';

comment on column t_report_claims.reply_time is
'回复时间';

comment on column t_report_claims.claims_mat_add_time is
'索赔资料提交时间';

comment on column t_report_claims.mat_return_date is
'发票寄回时间';

comment on column t_report_claims.close_date is
'结案日期';

comment on column t_report_claims.face_amount is
'发票金额';

comment on column t_report_claims.medi_assure_amount is
'医保统筹金额';

comment on column t_report_claims.third_pay_amount is
'第三方赔付金额';

comment on column t_report_claims.claim_amount is
'赔付金额';

comment on column t_report_claims.occurr_reason is
'出险原因';

comment on column t_report_claims.treatment_result is
'治疗结果';

comment on column t_report_claims.disease_diagnosis_pic is
'诊断证明';

comment on column t_report_claims.disability_certificate is
'残疾证明';

comment on column t_report_claims.death_certificate is
'死亡证明';

comment on column t_report_claims.student_status_certificate is
'学籍证明';

comment on column t_report_claims.refuse_desc is
'拒绝理由';

comment on column t_report_claims.domain_id is
'数据属主';

comment on column t_report_claims.creator is
'创建者用户ID';

comment on column t_report_claims.create_time is
'创建时间';

comment on column t_report_claims.updated_by is
'更新者';

comment on column t_report_claims.update_time is
'修改时间';

comment on column t_report_claims.addi is
'附加';

comment on column t_report_claims.remark is
'备注';

comment on column t_report_claims.status is
'状态:，2: 已报案，等待上传索赔资料，4: 受理中, 6: 等待补充资料, 8: 已结案, 10: 撤销报案, 12: 拒赔';

ALTER SEQUENCE t_report_claims_id_seq RESTART WITH 20000;


/*==============================================================*/
/* Table: t_resource                                            */
/*==============================================================*/
create table if not exists  t_resource (
   id                   SERIAL not null,
   insurance_type_id    INT8                 null,
   name                 VARCHAR              null,
   content              VARCHAR              null,
   link                 JSONB                null,
   picture              JSONB                null,
   tag                  VARCHAR              null,
   is_top               BOOL                 null,
   is_policy            BOOL                 null,
   updated_by           INT8                 null,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   creator              VARCHAR              null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   domain_id            INT8                 null,
   addi                 JSONB                null,
   remark               VARCHAR              null,
   status               VARCHAR              null,
   constraint PK_T_RESOURCE primary key (id)
);

comment on table t_resource is
'资源列表';

comment on column t_resource.id is
'资源编号';

comment on column t_resource.insurance_type_id is
'保险类型ID';

comment on column t_resource.name is
'资源名称';

comment on column t_resource.content is
'资源内容';

comment on column t_resource.link is
'链接';

comment on column t_resource.picture is
'图片';

comment on column t_resource.tag is
'标签';

comment on column t_resource.is_top is
'是否首页显示：用户进入智能客服后直接显示';

comment on column t_resource.is_policy is
'判断是否是保险条款';

comment on column t_resource.updated_by is
'更新者';

comment on column t_resource.update_time is
'更新时间';

comment on column t_resource.creator is
'创建者账号';

comment on column t_resource.create_time is
'创建时间';

comment on column t_resource.domain_id is
'数据属主';

comment on column t_resource.addi is
'附加数据';

comment on column t_resource.remark is
'备注';

comment on column t_resource.status is
'状态0:有效, 2:修改，4删除';

ALTER SEQUENCE t_resource_id_seq RESTART WITH 20000;

/*==============================================================*/
/* Table: t_scan_tdc                                            */
/*==============================================================*/
create table if not exists  t_scan_tdc (
   id                   SERIAL not null,
   tdc_id               INT8                 null,
   external_id          VARCHAR              null,
   req_time             INT8                 null,
   req_src              VARCHAR              null,
   constraint PK_T_SCAN_TDC primary key (id)
);

comment on table t_scan_tdc is
'请求二维码记录';

comment on column t_scan_tdc.id is
'二维码编号';

comment on column t_scan_tdc.tdc_id is
'二维码编号';

comment on column t_scan_tdc.external_id is
'外部平台ID';

comment on column t_scan_tdc.req_time is
'请求二维码时间';

comment on column t_scan_tdc.req_src is
'请求来源';

/*==============================================================*/
/* Table: t_school                                              */
/*==============================================================*/
create table if not exists  t_school (
   id                   SERIAL not null,
   name                 VARCHAR              not null,
   org_code             VARCHAR              null,
   faculty              JSONB                null,
   branches             JSONB                null,
   category             VARCHAR              not null,
   contact              VARCHAR              null,
   post_code            VARCHAR              null,
   phone                VARCHAR              null,
   addr                 VARCHAR              null,
   province             VARCHAR              null,
   city                 VARCHAR              null,
   district             VARCHAR              null,
   street               VARCHAR              null,
   data_sync_target     VARCHAR              null,
   sale_managers        JSONB                null,
   school_managers      JSONB                null,
   purchase_rule        JSONB                null,
   business_domain      VARCHAR              null,
   school_category      VARCHAR              null,
   allow_backdating     JSONB                null,
   use_credit_code      BOOL                 null default true,
   credit_code          VARCHAR              null,
   credit_code_pic      JSONB                null,
   invoice_title        VARCHAR              null,
   is_compulsory        BOOL                 null,
   reg_num              INT4                 null,
   compulsory_student_num INT8                 null,
   non_compulsory_student_num INT8                 null,
   dinner_num           INT4                 null,
   canteen_num          INT4                 null,
   shop_num             INT4                 null,
   files                JSONB                null,
   contact_role         VARCHAR              null,
   is_school            BOOL                 null,
   creator              INT8                 null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   updated_by           INT8                 null,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   domain_id            INT8                 null,
   addi                 JSONB                null,
   remark               VARCHAR              null,
   status               VARCHAR              null default '2',
   constraint PK_T_SCHOOL primary key (id)
);

comment on table t_school is
'学校信息表，包含了销售经理，学校管理员，投保规则';

comment on column t_school.id is
'学校编号';

comment on column t_school.name is
'名称';

comment on column t_school.org_code is
'机构代码';

comment on column t_school.faculty is
'学院';

comment on column t_school.branches is
'校区';

comment on column t_school.category is
'类别:幼儿园，小学，初中，高中，大学';

comment on column t_school.contact is
'联系人';

comment on column t_school.post_code is
'邮编';

comment on column t_school.phone is
'联系电话';

comment on column t_school.addr is
'详细地址';

comment on column t_school.province is
'省';

comment on column t_school.city is
'市';

comment on column t_school.district is
'区/县';

comment on column t_school.street is
'街道/片区';

comment on column t_school.data_sync_target is
'数据同步类型';

comment on column t_school.sale_managers is
'销售';

comment on column t_school.school_managers is
'学校管理员';

comment on column t_school.purchase_rule is
'投保规则';

comment on column t_school.business_domain is
'营业性质：文体体育、广告、事业单位、政府机关、其它';

comment on column t_school.school_category is
'学校性质：民办，公办';

comment on column t_school.allow_backdating is
'允许倒签';

comment on column t_school.use_credit_code is
'使用信用代码';

comment on column t_school.credit_code is
'统一社会信用代码';

comment on column t_school.credit_code_pic is
'统一社会信用代码证书，base64图片';

comment on column t_school.invoice_title is
'发票抬头';

comment on column t_school.is_compulsory is
'单位性质,true: 是义务教育，false: 不是非义务教育';

comment on column t_school.reg_num is
'注册人数';

comment on column t_school.compulsory_student_num is
'义务教育学生人数（校方）';

comment on column t_school.non_compulsory_student_num is
'非义务教育人数（校方）';

comment on column t_school.dinner_num is
'用餐人数';

comment on column t_school.canteen_num is
'食堂个数';

comment on column t_school.shop_num is
'商店个数';

comment on column t_school.files is
'附加文件';

comment on column t_school.contact_role is
'联系人职位';

comment on column t_school.is_school is
'是学校否';

comment on column t_school.creator is
'创建者用户ID';

comment on column t_school.create_time is
'创建时间';

comment on column t_school.updated_by is
'更新者';

comment on column t_school.update_time is
'更新时间';

comment on column t_school.domain_id is
'数据属主';

comment on column t_school.addi is
'附加数据';

comment on column t_school.remark is
'备注';

comment on column t_school.status is
'状态, ''0'': 未启用, ''2'': 启用, ''6'': 作废';

ALTER SEQUENCE t_school_id_seq RESTART WITH 20000;
-- select to_timestamp(cast(purchase_rule->>'Start' as bigint)/1000) from t_school;
-- select purchase_rule->0->'Rule'->>'End' from t_school;

insert into t_school(
  id,
  name,
  faculty,
  branches,
  category,
  data_sync_target,
  sale_managers,
  school_managers,
  purchase_rule,
  create_time
  ) values
  (
    1000,
    '回民小学',
    '[]',
    '[]',
    '小学',
    '微信',
    '[]',
    '[]',
    cast('[{"SN":0,"Start":' || (extract('epoch' from current_timestamp)*1000)::bigint 
						 || ', "End": ' || (extract('epoch' from current_timestamp)*1000 + 30*24*60*60*1000::bigint)::bigint 
						 || ',"TimeLimit": 6,"UnitPrice": 1,"InsuranceType": "学生意外伤害险-近邻"}]' as jsonb),
    (extract('epoch' from current_timestamp)*1000)::bigint
  ),(
    1002,
    '广大附中',
    '[]',
    '[]',
    '高中',
    '微信',
   '[]',
    '[]',
    cast('[{"SN":0,"Start":' || (extract('epoch' from current_timestamp)*1000)::bigint 
						 || ', "End": ' || (extract('epoch' from current_timestamp)*1000 + 30*24*60*60*1000::bigint)::bigint 
						 || ',"TimeLimit": 3,"UnitPrice": 1,"InsuranceType": "学生意外伤害险-校快保"}]' as jsonb),
    (extract('epoch' from current_timestamp)*1000)::bigint
  ),(
    1004,
    '广州大学',
    '[]',
    '[]',
    '大学',
    '微信',
    '[]',
    '[]',
    cast('[{"SN":0,"Start":' || (extract('epoch' from current_timestamp)*1000)::bigint 
						 || ', "End": ' || (extract('epoch' from current_timestamp)*1000 + 30*24*60*60*1000::bigint)::bigint 
						 || ',"TimeLimit": 4,"UnitPrice": 1,"InsuranceType": "学生意外伤害险-泰合"}]' as jsonb),
    (extract('epoch' from current_timestamp)*1000)::bigint
  ),(
    1006,
    '广外幼儿园',
    '[]',
    '[]',
    '幼儿园',
    '微信',
    '[]',
    '[]',
    cast('[{"SN":0,"Start":' || (extract('epoch' from current_timestamp)*1000)::bigint 
						 || ', "End": ' || (extract('epoch' from current_timestamp)*1000 + 30*24*60*60*1000::bigint)::bigint 
						 || ',"TimeLimit": 3,"UnitPrice": 1,"InsuranceType": "学生意外伤害险-近邻"}]' as jsonb),
    (extract('epoch' from current_timestamp)*1000)::bigint
  ),(
    1008,
    '华师附中',
    '[]',
    '[]',
    '初中',
    '微信',
    '[]',
    '[]',
    cast('[{"SN":0, "Start":' || (extract('epoch' from current_timestamp)*1000)::bigint 
						 || ', "End": ' || (extract('epoch' from current_timestamp)*1000 + 30*24*60*60*1000::bigint)::bigint 
						 || ',"TimeLimit": 3,"UnitPrice": 1,"InsuranceType": "学生意外伤害险-校快保"}]' as jsonb),
    (extract('epoch' from current_timestamp)*1000)::bigint
  ),(
    1010,
    '广州市花城小学',
    '[]',
   '[]',
    '小学',
    '微信',
    '[]',
    '[]',
    cast('[{"SN":0, "Start":' || (extract('epoch' from current_timestamp)*1000)::bigint 
						 || ', "End": ' || (extract('epoch' from current_timestamp)*1000 + 10*12*30*24*60*60*1000::bigint)::bigint 
						 || ',"TimeLimit": 3,"UnitPrice": 1,"InsuranceType": "学生意外伤害险-泰合"}]' as jsonb),
    (extract('epoch' from current_timestamp)*1000)::bigint
  ),(
    1012,
    '广州市乐成小学',
    '[]',
    '[]',
    '小学',
    '联保',
    '[]',
    '[]',
    cast('[{"SN":0, "Start":' || (extract('epoch' from current_timestamp)*1000)::bigint 
						 || ', "End": ' || (extract('epoch' from current_timestamp)*1000 + 10*12*30*24*60*60*1000::bigint)::bigint 
						 || ',"TimeLimit": 3,"UnitPrice": 1,"InsuranceType": "学生意外伤害险-近邻"}]' as jsonb),
    (extract('epoch' from current_timestamp)*1000)::bigint
  );
  
  update t_school set is_school=true;

/*==============================================================*/
/* Index: idx_school_name                                       */
/*==============================================================*/
create unique index if not exists  idx_school_name on t_school (
name
);

/*==============================================================*/
/* Table: t_section                                             */
/*==============================================================*/
create table if not exists  t_section (
   id                   SERIAL not null,
   name                 VARCHAR              not null,
   type                 VARCHAR              null,
   category             VARCHAR              null,
   issuer               VARCHAR              null,
   issue_time           INT8                 null,
   data                 JSONB                null,
   repo                 VARCHAR              not null,
   branch               VARCHAR              not null,
   repo_tag             VARCHAR              not null,
   tags                 JSONB                null,
   creator              INT8                 not null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   updated_by           INT8                 null,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   domain_id            INT8                 null,
   addi                 JSONB                null,
   remark               VARCHAR              null,
   status               VARCHAR              null,
   constraint PK_T_SECTION primary key (id)
);

comment on table t_section is
'the section/chapter of course';

comment on column t_section.id is
'编码';

comment on column t_section.name is
'名称';

comment on column t_section.type is
'类型';

comment on column t_section.category is
'分类';

comment on column t_section.issuer is
'制作者';

comment on column t_section.issue_time is
'发布时间';

comment on column t_section.data is
'附加数据';

comment on column t_section.repo is
'git repo';

comment on column t_section.branch is
'git repo branch';

comment on column t_section.repo_tag is
'git repo tag';

comment on column t_section.tags is
'标签';

comment on column t_section.creator is
'创建者';

comment on column t_section.create_time is
'创建时间';

comment on column t_section.updated_by is
'更新者';

comment on column t_section.update_time is
'更新时间';

comment on column t_section.domain_id is
'数据隶属';

comment on column t_section.addi is
'用户定制数据';

comment on column t_section.remark is
'备注';

comment on column t_section.status is
'enabled,有效
disabled,无效
expired,过期、无效';

/*==============================================================*/
/* Table: t_special_order                                       */
/*==============================================================*/
create table if not exists  t_special_order (
   id                   SERIAL not null,
   id_card_no           VARCHAR              not null,
   name                 VARCHAR              not null,
   grade                VARCHAR              null,
   district             VARCHAR              null,
   project              VARCHAR              not null,
   amount               FLOAT8               not null,
   pay_time             INT8                 null,
   open_id              VARCHAR              not null,
   trade_no             VARCHAR              null,
   transaction_id       VARCHAR              null,
   refund_no            VARCHAR              null,
   refund_time          INT8                 null,
   creator              INT8                 null,
   create_time          INT8                 null,
   updated_by           INT8                 null,
   update_time          INT8                 null,
   domain_id            INT8                 null,
   remark               VARCHAR              null,
   addi                 JSONB                null,
   status               VARCHAR              null,
   constraint PK_T_SPECIAL_ORDER primary key (id)
);

comment on table t_special_order is
't_special_order';

comment on column t_special_order.id is
'订单编号';

comment on column t_special_order.id_card_no is
'身份证号码';

comment on column t_special_order.name is
'姓名';

comment on column t_special_order.grade is
'年级';

comment on column t_special_order.district is
'校区';

comment on column t_special_order.project is
'收费项目';

comment on column t_special_order.amount is
'应收金额';

comment on column t_special_order.pay_time is
'支付时间';

comment on column t_special_order.open_id is
'open id';

comment on column t_special_order.trade_no is
'外部订单号';

comment on column t_special_order.transaction_id is
'支付平台订单号';

comment on column t_special_order.refund_no is
'退款单号';

comment on column t_special_order.refund_time is
'退款时间';

comment on column t_special_order.creator is
'创建者用户ID';

comment on column t_special_order.create_time is
'创建时间';

comment on column t_special_order.updated_by is
'更新者';

comment on column t_special_order.update_time is
'更新时间';

comment on column t_special_order.domain_id is
'数据属主';

comment on column t_special_order.remark is
'备注';

comment on column t_special_order.addi is
'附加数据';

comment on column t_special_order.status is
'状态,0：未支付，2：已支付，4：超时，6：作废';

ALTER SEQUENCE t_special_order_id_seq RESTART WITH 20000;

/*==============================================================*/
/* Index: special_order_prj_id                                  */
/*==============================================================*/
create unique index if not exists  special_order_prj_id on t_special_order (
id_card_no,
project
);

/*==============================================================*/
/* Index: special_order_open_id                                 */
/*==============================================================*/
create  index if not exists  special_order_open_id on t_special_order (
open_id
);

/*==============================================================*/
/* Table: t_student_answers                                     */
/*==============================================================*/
create table if not exists  t_student_answers (
   id                   INT4                 not null,
   type                 VARCHAR(128)         null,
   examinee_id          INT8                 null,
   practice_submission_id INT8                 null,
   question_id          INT8                 null,
   answer               JSONB                not null,
   answer_score         FLOAT8               null,
   marker               JSONB                null,
   "order"              INT4                 null,
   group_id             INT8                 null,
   actual_options       JSONB                null,
   actual_answers       JSONB                null,
   wrong_attempt        INT4                 null,
   answer_attach        JSONB                null,
   creator              INT8                 not null,
   create_time          INT8                 null,
   updated_by           INT8                 null,
   update_time          INT8                 null,
   addi                 JSONB                null,
   status               VARCHAR(64)          null default '0',
   constraint PK_T_STUDENT_ANSWERS primary key (id, answer)
);

comment on table t_student_answers is
'考卷题目得分表';

comment on column t_student_answers.id is
'编号';

comment on column t_student_answers.type is
'类型, 00:考试  02:练习';

comment on column t_student_answers.examinee_id is
'学生考试ID';

comment on column t_student_answers.practice_submission_id is
'学生练习ID';

comment on column t_student_answers.question_id is
'考卷题目ID';

comment on column t_student_answers.answer is
'学生答案';

comment on column t_student_answers.answer_score is
'学生答案得分';

comment on column t_student_answers.marker is
'题目批阅信息';

comment on column t_student_answers."order" is
'题目排序';

comment on column t_student_answers.group_id is
'所在题组ID';

comment on column t_student_answers.actual_options is
'实际题目的选项';

comment on column t_student_answers.actual_answers is
'实际题目客观题答案';

comment on column t_student_answers.wrong_attempt is
'进入错题集的第n次练习答题';

comment on column t_student_answers.answer_attach is
'考试附件路径';

comment on column t_student_answers.creator is
'创建者';

comment on column t_student_answers.create_time is
'创建时间';

comment on column t_student_answers.updated_by is
'更新者';

comment on column t_student_answers.update_time is
'更新时间';

comment on column t_student_answers.addi is
'附加信息';

comment on column t_student_answers.status is
'数据状态';

/*==============================================================*/
/* Table: t_sys_ver                                             */
/*==============================================================*/
create table if not exists  t_sys_ver (
   id                   SERIAL not null,
   tag                  VARCHAR              null,
   name                 VARCHAR              null,
   ver                  VARCHAR              null,
   create_time          VARCHAR              null,
   update_time          VARCHAR              null,
   addi                 JSONB                null,
   remark               VARCHAR              null,
   status               VARCHAR              null,
   constraint PK_T_SYS_VER primary key (id)
);

comment on table t_sys_ver is
'应用版本包含业务模型、前端、后端、配置文件等

1、业务模型版本在模型生成时建立；
2、后端模型版本在每次后端启动时建立或更新；
3、配置文件版本在每次后端启动时建立或更新；
4、前端版本在每次后端启动时建立或更新；

';

comment on column t_sys_ver.id is
'编码';

comment on column t_sys_ver.tag is
'标识';

comment on column t_sys_ver.name is
'名称';

comment on column t_sys_ver.ver is
'版本';

comment on column t_sys_ver.create_time is
'创建时间';

comment on column t_sys_ver.update_time is
'更新时间';

comment on column t_sys_ver.addi is
'附加';

comment on column t_sys_ver.remark is
'备注';

comment on column t_sys_ver.status is
'状态';

ALTER SEQUENCE t_sys_ver_id_seq RESTART WITH 20000;

insert into t_sys_ver(id,name,ver,create_time,update_time,remark)
  values(1000,'业务模型','3.1.10.0',
  '2016年12月5日 9:52:53','2025年8月18日 16:29:26',
  '3.1.10.0
优化练习错题集视图SQL生成语句 优化practice_summary 视图attempt_count字段获取方式

3.1.9.0
优化试卷视图区分发布前后查询，添加试卷表exampaper_id及version字段

3.1.8.0
优化考卷视图查询语句 、 新增练习错题集视图 、增加学生作答表与练习提交表 错题练习次数 用于错题集的题目提取

3.1.7.0
增加视图v_exam_file，为t_examinee补充exam_paper_id字段

3.1.6.0
增加考试信息表、考试场次表和考生表之间的外键依赖，修改v_examinee_info视图中对于灵活时段考试方式的actual_end_time计算方式

3.1.5.0
去除题库表题目数量字段，建立题库与题库题目、试卷题目与题库题目、试卷和试卷题组和试卷题目的外键，去除题库共享表，添加v_question_bank视图

3.1.4.0
增加考卷、考卷题组、考卷题目、学生答卷外键 + 级联删除 修改题库、题目、共享题目表关于时间字段的属性为int8

3.1.3.0
优化v_paper视图逻辑，添加v_z_grade_exam_session_info和v_z_grade_practice_statistics,删除v_paper_share和t_resource_share表,补充v_student_practice_total_score的usedtime字段

3.1.2.0
更改v_paper 与v_exam_paper视图生成逻辑

3.1.1.0
为v_examinee_info添加serial_num,以及为t_exam_site添加了domain_id和sys_user

3.1.0.0
添加考试系统的表和视图

3.0.0.1
重构表t_user/t_domain/t_user_doman的初始化数据

2.8.2.1
为t_course/section添加tags

2.8.2
为t_my_contact, t_msg_status添加主键

2.8.1
添加t_user_group

2.8.0
添加了即时通信相关表，更改了原来的t_msg为t_article

2.7.2
更改了t_relation表，增加了right_key
remove all jsonb data type default value

2.7.1.2
 更新健康告知书

2.7.1.1
  alter t_v_order/order2 with limit 1

2.7.1 add health_survey to v_order/v_order2

2.7
  add t_domain.level for distinguish domain 

2.6.5.1 
    add v_insure_attach.insurer/insurance_type

2.6.5
  add t_relation.right_value_type,right_value

2.6.4.2 
  残疾=>疾病

2.6.4.1
    add v_insurance_type.other_files

2.6.4
   add t_insurance_types.other_files jsonb
2.6.3
  add v_insurer


2.6.2.1
refine index t_import_data.entity_id

2.6.2
add t_import_data.entity_id
add t_order.agency_id

2.6.1
add t_import_data.file_digest

2.6.0
add t_import_data

2.5.18
add v_user_domain_api

2.5.17
synchronize t_relation_history with t_relation

2.5.16
add v_domain_api.access_control_level

2.5.15.5
refine v_report_claims.insurance_type_id

2.5.15.4
 add idx for t_insurance_policy

2.5.15.3
v_insurance_policy, v_insurance_policy2,v_insure_attach添加上传文件状态

2.5.15.2
  refine plan_type

2.5.15.1
  add idx_user_official_name
  refine v_order2.plan_type

2.5.15
 add play_type to v_order2

2.5.14.1
minor refine

2.5.14
restore v_insurance_policy, v_insure_attach to improve performance, v_insurance_policy2



2.5.13
refine v_insurance_policy, v_insure_attach to improve performance, v_insurance_policy2

2.5.12.1
trivial

2.5.12
v_insurance_policy,v_report_claims add o.plan_id
add t_order.refound_confirm

2.5.11
 add t_order/v_order2..pay_name

2.5.10.4
refine t_insurance_types intialization data.


2.5.10.3
refine t_insurance_types intialization data.

2.5.10.2
add v_insurance_type.data_type

2.5.10.1
change v_insurance_type data filter condition to data_type=04 or 06 or 08


2.5.10
add t_mistake_correct/v_mistake_correct2..need_blance

2.5.9.1
v_mistake_correct_show, v_order_2


2.5.9
add t_file.file_oid, v_domain_api.domain


2.5.8.4
add v_mistake_correct2.have_dinner_num


2.5.8.3
alter v_insurance_policy, v_insurance_policy2

2.5.8.2
rebuild v_insure_attach

2.5.8.1
add v_insurance_policy2.plan_id

2.5.8
refine table common fields: regenerator

2.5.7.1
alter v_insurance_policy

2.5.7 
 add t_school.addi

2.5.6.6
refine t_insurance_types, t_param init data

2.5.6.5
refine v_insurance_type

2.5.6.4
redefine v_mistake_correct_show

2.5.6.3
redefine v_mistake_correct_show

2.5.6.2
update v_user_domain

2.5.6.1
update v_insurance_policy,v_insure_attach


2.5.6
add v_mistake_correct_show

2.5.5.2
v_mistakecorrect2.plan_name

2.5.5.1
add v_mistake_correct2.insurer

2.5.5
add t_mistakecorrect.plan_id

2.5.4
add t_insurance_types.transfer_auth_files
add v.insurance_type..transfer_auth_files
alter v.insurance_type.receipt_account, contact_qr_code

2.5.3.2
change 学校年、班初始化数据, 添加未分班、完中的支持

2.5.3.1
change initial data

2.5.3
add v_insurance_type.time_status

2.5.2
remove default json value from t_user_domain,t_domain_api

2.5.1
refine v_insured_school

2.5
add view v_insured_school

2.4.9
add t_user_domain.grant_source,data-access_mode,data_scope

2.4.8
add t_school.allow_backdating

2.4.7.2
add data 健康告知书 to t_param(id,belongto) values(14000,11000

2.4.7.1
o.plan_id,o.plan_name,o.insurer,


2.4.7
add t_order.plan_name, insurer

2.4.6.4
refine v_insurance_type.insured_start_time

2.4.6.3
add unit_price initial data to t_insurance_types 

2.4.6.2
add v_order.charge_mode

2.4.6.1
update t_insurance_types initial data.

2.4.6
add t_insurance_types.unit_price
add v_insurance_type.unit_price


2.4.5.2
alter v_order2
alter t_insurance_types initialize data.


2.4.5.1
insert into t_param(id,belongto,name,value) values(13436,12036,草稿,10000,10020);


2.4.5
add v_insurance_type.batch

2.4.4
add t_insurace_types.resource
add v_insurance_type.resource,create_time,update_time

2.4.3
add v_insurance_type.status

2.4.2
 add t_special_order

2.4.1 
  redesign v_insurance_type

2.4
add t_insurace_types.ref_id 用于表达机构引用的保险方案编号

2.3
add t_order.plan_snap 保险方案快照，用于保存订单使用的保险方案

2.2
add t_order.health_survey

2.1.1
add v_insurance_type

2.1
add t_order.plan_id, t_insurance_types.insurer
add test data for t_insurance_types

2.0.2 
add index to t_region.region_name
alter v_region define


2.0.1
add t_order.age_limt


2.0
作废 t_age
add t_order.street
add t_insurance_types many fields

1.0.58.1

insert into t_param(id,belongto,name) values(12090,11000,智能客服
insert into t_param(id,belongto,name,value) values(13930,12090,联系电话


1.0.58
add t_resource.link, t_resource.picture

1.0.57.1
add o.balance, o.balance_list to v_mistake_correct2

1.0.57
add t_mistake_correct.revoked_policy_no


1.0.56
add t_mistake_correct.have_negotiated_price


1.0.55.3
t_external_domain_conf.status default value set to 02

1.0.55.2
t_param add parameters

1.0.55.1
v_order2

1.0.55
t_black_list.creat_time alter with create_time

1.0.54
t_insurance_policy.favorite
t_report_claims.refuse_desc

1.0.53.2
o.order_status

1.0.53.1
update t_price initialize data.

1.0.53
t_order.balance, balance_list

1.0.52
t_mistake_correct


1.0.51
v_order2 altered

insert into t_param(id,belongto,name,value) values(13360,12026,...

1.0.50
t_mistake_correct.
    files_to_remove, 
    clear_list,
    policy_regen


1.0.49.3
v_order2
t_school.use_credit_code


1.0.49.2
v_order2
insured_post_code,insured_phone,policy_scheme_title


1.0.49.1
v_insurance_policy2

1.0.49
t.school.use_credit_code
1.0.48
t_param
t_price
t_insurance_types
v_order2
v_mistake_correct2

1.0.47.1
t_param intialize data alter

1.0.47
t_msitake_correct.correct_level
v_mistake_correct.correct_level


1.0.46
t_msitake_correct
v_mistake_correct

1.0.45
v_order2



1.0.44
v_mistake_correct2.have_insured_list

1.0.43.2
v_order2.correct_times

1.0.43.1 
v_order2

1.0.43
v_insurance_policy2


1.0.42
t_insurance_policy add missing column from t_order

1.0.41
add t_insurace_detail.field_type

add initialize data to t_param



1.0.40
t_report_claims.policy_file, insurance_type_id
v_report_claims

1.0.39
    t_param
    t_param.sql     ﻿
    t_order
    v_order2
    t_insured_detail
    t_price

    t_report_claims, v_report_claims


1.0.38
add t_insurance_policy.have_sudden_death


1.0.37.1
alter t_external_domain_conf.status default value to 01

1.0.37

t_insurance_types
添加实习生险议价联系人
v_order2
t_param
添加转账授权说明路径和标签

v_mistake_correct2


1.0.36
t_report_claims(报案表)

增加字段
occurr_reason varchar 出险原因
treatment_result varchar 治疗结果
disease_diagnosis_pic jsonb 疾病诊断证明
disability_certificate jsonb 残疾证明资料
death_certificate jsonb 身故证明资料
 student_status_certificate jsonb 学籍证明

v_report_claims(报案视图)
1.增加字段 

i.sn,
r.occurr_reason,
r.treatment_result,
r.disease_diagnosis_pic,
r.disability_certificate,
r.death_certificate,
r.student_status_certificate

1.0.35
t_insured_detail
添加列
is_open 是否开放 bool
is_heated 是否恒温 bool
is_training 是否用作训练 bool
province 省 varchar
city 市 varchar
district 区 varchar
addr 详细地址 varchar
train_item 训练项目(英文逗号分隔) varchar
other_item 其他项目(英文逗号分隔) varchar
area 场地面积 float

t_insurance_types
字段修改: sudden_death_description varchar类型改为jsonb，还要分类型= =

初始化数据全部替换:

t_param
添加参数-议价类型



1.0.34
t_price
全部替换
t_order
添加列
是否开启猝死责任险 have_sudden_death bool
场地个数  ground_num int

并补充到v_order2视图中
o.have_sudden_death,
o.ground_num ,

t_insurance_types
添加列 sudden_death_description 猝死责任险描述 jsonb

全部替换:
t_negotiated_price
添加列 location 地点 varchar
基于insurance_type_id，commence_date， location 三列建立索引

1.0.33
v_order2视图
添加列 o.reminders_num 

t_insurance_types
添加列 files jsonb

v_insurance_policy2

1.0.32
t_insurance_types
添加列 list_tpl 清单模板 varchar类型(已添加)

t_price

v_mistake_correct2
视图修改
t_param
全部替换

v_order2
添加列

1.0.31
add
t_insurance_types.list_tpl

1.0.30
t_insurance_types 
添加列 interval 时间间隔 int8

t_price 
添加列 title 标题 varchar
添加列 category 类型 varchar

t_param
 init value

1.0.29
t_age, 替换初始化数据
添加列 insured_count int64 被保险人数

v_mistake_correct2
添加insured_count 
提取投保机构/被保险机构字段用于导出excel

1.0.28
    t_order
    v_order2
    t_insured_detail
    v_mistake_correct2
    t_price
    t_param


1.0.27
add
t_insurance_types.external_status



1.0.26
add
t_insurance_types.alias

1.0.25
v_insurance_policy2(二期保单视图)
删除字段 i.org_id
增加字段o.org_id

t_order
添加 have_dinner_num bool 是否开启就餐人数

v_mistake_correct2
提取投保单位名方便查询
o.policyholder->>Name as org_name,

1.0.24
t_mistake_correct
添加列 pay_type 支付方式
添加列 fee_scheme 计费方式

v_mistake_correct2
添加新增列

1.0.23
t_insurance_policy/v_insurance_policy2
cancel_desc
zero_pay_status

t_price
template_path varchar => files jsonb

1.0.22.1
v_insurance_policy2

1.0.22
t_param
调整校方系统参数

t_insurance_types
添加校方对公转账信息

v_order2
添加座位数的统计

t_price
添加列 template_path varchar 模板路径 

1.0.21.1
    refine

1.0.21
    t_param
    t_mistake_correct
    v_mistake_correct2
    v_order2

1.0.20
t_price,v_order2,t_mistake_correct,v_mistake_correct2

1.0.19
    t_insurance_types表
    t_order表
    t_price表
    t_insure_detail表
    v_mistake_correct2视图

1.0.18.2
t_v_payment


1.0.18.1
  add t_msg

1.0.18
  add v_order2.have_policy
  some trial change

1.0.17


1.0.16.1
# v_insurance_policy2(二期保单视图)
增加字段
o.charge_mode
i.third_party_account

1.0.16
t_insure_attach
增加 insure_policy_id, 对应t_insurance_policy.id

v_insurance_policy2
增加 a.insure_policy_id, 

1.0.15
t_age表
设置status默认值为0

t_param表
修改学意险默认价格为10000

添加地区选择器初始化地区
新增 t_payment 表
t_payment 缴费表(用于对公转账自动化)

1.0.14
t_insurance_types表
初始化数据修改:修改校方责任险初始化数据，修改比赛活动保险首页显示最低价
附件: t_insurance_types.sql https://uploader.shimo.im/f/PaCoMFK1mDoTcGPA.sql

t_mistake_correct表
添加列 indate 保险期间 int8
添加列 charge_mode 计费方式 varchar

t_age表
列 enabled 的默认值设置为true

t_param表
添加校方责任险所需系统参数
附件: t_param.sql https://uploader.shimo.im/f/99Dsy6FmM1oReerm.sql

v_mistake_correct视图(保留一期的内容不变)

v_mistake_correct2视图
-- 供二期更正使用,未兼容一期



1.0.13 
  2020.03.06
add v_mistake_correct2

1.0.12 
    2020.03.04

1.0.10
增加权限视图

1.0.8
t_price 初始化数据修改：去掉协议价标准
t_insurance_types表  添加健康险网页描述
t_param表 添加系统参数: 协议价标准
t_report_claims表删除初始化数据

*********************************************
1.0.7
t_price 初始化数据修改：比赛险的价格方案添加"ReqStandard":3000
t_insurance_types表     删除原description列
添加列 web_description 网页描述 varchar
          mobile_description 移动端描述 varchar
初始化数据添加健康险及各险种描述
t_param表 初始化数据添加校方支付方式
*********************************************
1.0.6 add t_insurance_types.enable_import_list
*********************************************
1.0.4 更新 t_insurance_types,t_param 初始化数据
*********************************************
1.0.3更新v_order2.policy_no
**********************************************
请每次更新时以SemanticVersion修改版本号
即a.b.c
a增加, 表示修改原有接口导致不兼容变更
b增加, 表示引入新功能，但与之前接口兼容
c增加, 表示修正bugs未增加新功能，并与之前兼容');

/*==============================================================*/
/* Table: t_tdc                                                 */
/*==============================================================*/
create table if not exists  t_tdc (
   id                   SERIAL not null,
   tdc_id               VARCHAR              null,
   name                 VARCHAR              null,
   issuer               INT8                 not null,
   issue_time           INT8                 null,
   limn                 VARCHAR              null,
   data                 JSONB                null,
   expiration           INT8                 null,
   type                 VARCHAR              null,
   goto_view            VARCHAR              null,
   requested            integer              null,
   accepted             INT2                 null,
   remark               VARCHAR              null,
   status               VARCHAR              null,
   constraint PK_T_TDC primary key (id)
);

comment on table t_tdc is
'two-dimension-code';

comment on column t_tdc.id is
'编码';

comment on column t_tdc.tdc_id is
'二维码标识';

comment on column t_tdc.name is
'名称';

comment on column t_tdc.issuer is
'发布者';

comment on column t_tdc.issue_time is
'发布时间';

comment on column t_tdc.limn is
'描述';

comment on column t_tdc.data is
'附加数据';

comment on column t_tdc.expiration is
'过期时间';

comment on column t_tdc.type is
'类型';

comment on column t_tdc.goto_view is
'扫描后的目标页面';

comment on column t_tdc.requested is
'使用次数';

comment on column t_tdc.accepted is
'成功使用次数';

comment on column t_tdc.remark is
'备注';

comment on column t_tdc.status is
'enabled,有效
disabled,无效
expired,过期、无效';

ALTER SEQUENCE t_tdc_id_seq RESTART WITH 10000;

/*==============================================================*/
/* Table: t_teacher_student                                     */
/*==============================================================*/
create table if not exists  t_teacher_student (
   id                   SERIAL               not null,
   creator              INT8                 not null,
   create_time          TIMESTAMP            null,
   updated_by           INT8                 null,
   update_time          TIMESTAMP            null,
   status               VARCHAR(150)         null,
   addi                 JSONB                null,
   constraint PK_T_TEACHER_STUDENT primary key (id)
);

comment on table t_teacher_student is
'学生教师关联表';

comment on column t_teacher_student.id is
'表主键ID';

comment on column t_teacher_student.creator is
'创建者';

comment on column t_teacher_student.create_time is
'创建时间';

comment on column t_teacher_student.updated_by is
'更新者';

comment on column t_teacher_student.update_time is
'更新时间';

comment on column t_teacher_student.status is
'状态 00：正常  02：异常';

comment on column t_teacher_student.addi is
'附加信息';

/*==============================================================*/
/* Table: t_undertaker                                          */
/*==============================================================*/
create table if not exists  t_undertaker (
   id                   SERIAL not null,
   prj_id               INT8                 null,
   developer_id         INT8                 null,
   developer_type       VARCHAR              null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   remark               VARCHAR              null,
   status               VARCHAR              null,
   constraint PK_T_UNDERTAKER primary key (id)
);

comment on table t_undertaker is
'项目承接表';

comment on column t_undertaker.id is
'承接编号';

comment on column t_undertaker.prj_id is
'项目编号';

comment on column t_undertaker.developer_id is
'开发者编号';

comment on column t_undertaker.developer_type is
'undertaker,承接者
invitee,邀请承接
apply,申请承接';

comment on column t_undertaker.create_time is
'创建时间';

comment on column t_undertaker.update_time is
'更新时间';

comment on column t_undertaker.remark is
'备注';

comment on column t_undertaker.status is
'状态
inviting,邀请承接中
applying,申请承接中
accepted,已接受
rejected,被拒绝
signed, 已签定合同';

/*==============================================================*/
/* Table: t_user_assessment                                     */
/*==============================================================*/
create table if not exists  t_user_assessment (
   id                   SERIAL not null,
   user_id              INT8                 not null,
   exam_id              INT8                 null,
   paper_id             INT8                 null,
   examiner_id          INT8                 null,
   reviewer_id          INT8                 null,
   test_items_id        INT8                 not null,
   score                NUMERIC              null,
   scored               NUMERIC              not null,
   answer_type          VARCHAR              null,
   answer               VARCHAR              null,
   answering            VARCHAR              null,
   feedback             VARCHAR              null,
   msg                  VARCHAR              null,
   addi                 JSONB                null,
   creator              INT8                 null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   updated_by           INT8                 null,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   remark               VARCHAR              null,
   status               VARCHAR              null,
   constraint PK_T_USER_ASSESSMENT primary key (id)
);

comment on table t_user_assessment is
'学生评价管理';

comment on column t_user_assessment.id is
'作答id';

comment on column t_user_assessment.user_id is
'考生编号';

comment on column t_user_assessment.exam_id is
'考试编号';

comment on column t_user_assessment.paper_id is
'试卷编号';

comment on column t_user_assessment.examiner_id is
'阅卷人';

comment on column t_user_assessment.reviewer_id is
'审核 人';

comment on column t_user_assessment.test_items_id is
'题目编号';

comment on column t_user_assessment.score is
'题目分数';

comment on column t_user_assessment.scored is
'本题得分';

comment on column t_user_assessment.answer_type is
'答案类型,文本, 多媒体';

comment on column t_user_assessment.answer is
'正确答案';

comment on column t_user_assessment.answering is
'考生作答';

comment on column t_user_assessment.feedback is
'评阅意见';

comment on column t_user_assessment.msg is
'检测过程信息';

comment on column t_user_assessment.addi is
'用户定制数据';

comment on column t_user_assessment.creator is
'创建者';

comment on column t_user_assessment.create_time is
'创建时间';

comment on column t_user_assessment.updated_by is
'更新者';

comment on column t_user_assessment.update_time is
'更新时间';

comment on column t_user_assessment.remark is
'备注';

comment on column t_user_assessment.status is
'可用，禁用';

/*==============================================================*/
/* Table: t_user_course                                         */
/*==============================================================*/
create table if not exists  t_user_course (
   id                   SERIAL not null,
   u_id                 INT8                 not null,
   c_id                 INT8                 not null,
   not_before           INT8                 null,
   not_after            INT8                 null,
   sections_sum_digest  VARCHAR              not null,
   sections             JSONB                not null,
   sections_sync_time   INT8                 null,
   score                JSONB                null,
   learn_status         JSONB                null,
   creator              INT8                 not null,
   create_time          INT8                 not null default (extract(epoch from current_timestamp)*1000)::bigint,
   updated_by           INT8                 null,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   domain_id            INT8                 null,
   addi                 JSONB                null,
   status               VARCHAR              not null,
   constraint PK_T_USER_COURSE primary key (id)
);

comment on table t_user_course is
'用户课程表';

comment on column t_user_course.id is
'关系编号';

comment on column t_user_course.u_id is
'用户编号';

comment on column t_user_course.c_id is
'课程编号';

comment on column t_user_course.not_before is
'允许使用开始时间';

comment on column t_user_course.not_after is
'允许使用结束时间';

comment on column t_user_course.sections_sum_digest is
'课程目录数字摘要';

comment on column t_user_course.sections is
'课程目录快照';

comment on column t_user_course.sections_sync_time is
'课程目录同步时间';

comment on column t_user_course.score is
'成绩';

comment on column t_user_course.learn_status is
'学习状态';

comment on column t_user_course.creator is
'创建者';

comment on column t_user_course.create_time is
'创建时间';

comment on column t_user_course.updated_by is
'更新者';

comment on column t_user_course.update_time is
'更新时间';

comment on column t_user_course.domain_id is
'数据隶属';

comment on column t_user_course.addi is
'用户定制数据';

comment on column t_user_course.status is
'00: 关注/收藏
02: 试学
04: 已购买、可退款
06: 学习进度超过退款范围
08: 完成学习
10: 完成课程期末考试
12: 取消收藏
14: 退款
16: 例外退款';

/*==============================================================*/
/* Index: idx_user_course_u_c                                   */
/*==============================================================*/
create unique index if not exists  idx_user_course_u_c on t_user_course (
u_id,
c_id
);

/*==============================================================*/
/* Table: t_user_degree                                         */
/*==============================================================*/
create table if not exists  t_user_degree (
   id                   SERIAL not null,
   user_id              INT8                 null,
   degree_id            INT8                 null,
   constraint PK_T_USER_DEGREE primary key (id)
);

comment on table t_user_degree is
'用户等级表';

comment on column t_user_degree.id is
'用户能力等级编号';

comment on column t_user_degree.user_id is
'用户编号';

comment on column t_user_degree.degree_id is
'能力等级编号';

/*==============================================================*/
/* Table: t_user_domain                                         */
/*==============================================================*/
create table if not exists  t_user_domain (
   id                   SERIAL not null,
   sys_user             INT8                 null,
   id_on_domain         VARCHAR              null,
   domain               INT8                 null,
   grant_source         VARCHAR              null,
   data_access_mode     VARCHAR              null,
   data_scope           JSONB                null,
   domain_id            INT8                 null,
   creator              INT8                 null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   updated_by           INT8                 null,
   addi                 JSONB                null,
   remark               VARCHAR              null,
   status               VARCHAR              null default '01',
   constraint PK_T_USER_DOMAIN primary key (id)
);

comment on table t_user_domain is
'用户，组，角色配置表';

comment on column t_user_domain.id is
'编码';

comment on column t_user_domain.sys_user is
'系统用户编码';

comment on column t_user_domain.id_on_domain is
'基于用户域的用户编码，如广州大学员工号，后勤部员工号，采购组采购员编号，保卫科保安员工号';

comment on column t_user_domain.domain is
'域, 格式: 机构[部门.科室.组[^角色[!userID]]]，[option]表示可选';

comment on column t_user_domain.grant_source is
'grant:数据权限由t_relation中left_type:t_domain.id与left_type:t_user.id获得的数据决定,或data_scope中数据决定，但data_scope与t_relation只能存在一种，如果data_scope有效，则忽略t_relation;

cousin:忽略data_scope与t_relation, 授权数据由被过虑数据的domain_id决定,即被过虑数据的domain_id 与登录用户的t_user.domain_id相同或级别更低的数据，例如
    用户的t_user.domain为xkb^admin而数据的domain为xkb.school^admin，则用户可以获得该数据

self: 被过虑数据的creator 与登录用户的t_user.id相同

api: 由功能(api)自己决定 ';

comment on column t_user_domain.data_access_mode is
'数据访问类型, full:可读写, read: 只读, write: 写, partial: 部分写/混合';

comment on column t_user_domain.data_scope is
'当grant_source是grant时,以json数据方式提供数据授权范围格式为:
  {"granter":"t_user.id","grantee":"t_school.id","data":[1234,456,789]}
granter: 代表数据拥有者, t_user.id代表用户, t_domain.id代表角色,t_api.id代表功能
grantee: 代表拥有的数据,t_school.id代表可以访问的机构列表。
授权数据如果存储在t_relation中则各项分别对应如下
granter对应left_type, left_key对应t_user_domain.sys_user或t_domain_api.domain
grantee对应right_type, right_key对应right_type的意义';

comment on column t_user_domain.domain_id is
'数据隶属';

comment on column t_user_domain.creator is
'本数据创建者';

comment on column t_user_domain.create_time is
'生成时间';

comment on column t_user_domain.update_time is
'帐号信息更新时间';

comment on column t_user_domain.updated_by is
'更新者';

comment on column t_user_domain.addi is
'附加信息';

comment on column t_user_domain.remark is
'备注';

comment on column t_user_domain.status is
'状态，00：草稿，01：有效，02：作废';

ALTER SEQUENCE t_user_domain_id_seq RESTART WITH 20000;

drop trigger if exists trigger_user_domain on t_user_domain;
drop function if exists user_domain_sync cascade;


create or replace function user_domain_sync()
returns trigger
as $$
begin

case 
-- TG_OP: trigger operation
when TG_OP = 'INSERT' then 
	update t_user set role=new.domain where id=new.sys_user;
	
when TG_OP = 'UPDATE' then
	update t_user set role=new.domain where id=new.sys_user;
	
when TG_OP = 'DELETE' then
-- 	raise notice 'on %', TG_OP;
	update t_user 
		set role=(
			select domain from t_user_domain 
				where sys_user=old.sys_user
				order by create_time desc,domain
			  limit 1
		) 
		where id=old.sys_user;
	
end case;
return NULL;
end;
$$ language plpgsql;

create trigger trigger_user_domain_del after insert or update or delete
on t_user_domain
for each row
execute function user_domain_sync();



insert into t_user_domain(sys_user,domain,domain_id,creator) values
(1000,333,322,1000),
(1000,366,322,1000),
(1000,566,322,1000),
(1000,1079,322,1000),
(1000,10100,322,1000),
(1002,366,322,1000),
(1002,566,322,1000),
(1002,10100,322,1000),
(1002,10102,322,1000),
(1002,10104,322,1000),
(1002,10106,322,1000),
(1004,10102,322,1000),
(1008,10104,322,1000),
(1008,10106,322,1000),
(1010,366,322,1000),
(1010,566,322,1000),
(1010,10100,322,1000),
(1010,10102,322,1000),
(1010,10104,322,1000),
(1010,10106,322,1000),
(1111,10104,322,1000),
(1212,10108,322,1000),

(1313,10100,322,1000),
(1313,10102,322,1000),
(1313,10104,322,1000),
(1313,10106,322,1000),
(1313,10108,322,1000),
(1313,10202,322,1000),
(1313,10204,322,1000),
(1313,10208,322,1000),
(1313,10210,322,1000),
(1313,10212,322,1000),
(1313,10214,322,1000),


(1404,10202,322,1000),
(1406,10204,322,1000),
(1408,10208,322,1000),
(1410,10210,322,1000),
(1412,10212,322,1000),
(1414,10214,322,1000),
(1416,10214,322,1000),
(1418,10214,322,1000);

/*==============================================================*/
/* Index: idx_user_domain                                       */
/*==============================================================*/
create unique index if not exists  idx_user_domain on t_user_domain (
domain,
sys_user
);

/*==============================================================*/
/* Table: t_user_group                                          */
/*==============================================================*/
create table if not exists  t_user_group (
   id                   SERIAL not null,
   user_id              INT8                 not null,
   group_id             INT8                 not null,
   domain_id            INT8                 null,
   creator              INT8                 null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   updated_by           INT8                 null,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   addi                 JSONB                null,
   remark               VARCHAR              null,
   status               VARCHAR              null default '01',
   constraint PK_T_USER_GROUP primary key (id)
);

comment on table t_user_group is
'user belong to group';

comment on column t_user_group.id is
'关系编号';

comment on column t_user_group.user_id is
'用户';

comment on column t_user_group.group_id is
'组';

comment on column t_user_group.domain_id is
'数据隶属';

comment on column t_user_group.creator is
'本数据创建者';

comment on column t_user_group.create_time is
'生成时间';

comment on column t_user_group.updated_by is
'更新者';

comment on column t_user_group.update_time is
'帐号信息更新时间';

comment on column t_user_group.addi is
'附加信息';

comment on column t_user_group.remark is
'备注';

comment on column t_user_group.status is
'状态，00：草稿，01：有效，02：作废';

/*==============================================================*/
/* Index: idx_user_grp                                          */
/*==============================================================*/
create unique index if not exists  idx_user_grp on t_user_group (
user_id,
group_id
);

/*==============================================================*/
/* Table: t_wx_user                                             */
/*==============================================================*/
create table if not exists  t_wx_user (
   id                   INT8                 not null,
   Subscribe            INT4                 null,
   Subscribe_Time       INT4                 null,
   wx_open_id           VARCHAR              null,
   mp_open_id           VARCHAR              null,
   pay_open_id          VARCHAR              null,
   union_ID             VARCHAR              null,
   Group_ID             INT4                 null,
   open_id              VARCHAR              null,
   Tag_ID_List          VARCHAR              null,
   Nickname             VARCHAR              null,
   Sex                  INT4                 null,
   Language             VARCHAR              null,
   City                 VARCHAR              null,
   Province             VARCHAR              null,
   Country              VARCHAR              null,
   Head_img_URL         VARCHAR              null,
   Privilege            VARCHAR              null,
   QR_Scene             INT4                 null,
   Subscribe_Scene      VARCHAR              null,
   QR_Scene_Str         VARCHAR              null,
   Err_Code             INT4                 null,
   Err_Msg              VARCHAR              null,
   creator              INT8                 null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   updated_by           INT8                 null,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   domain_id            INT8                 null,
   Remark               VARCHAR              null,
   addi                 JSONB                null,
   status               VARCHAR              null default '01',
   constraint PK_T_WX_USER primary key (id)
);

comment on table t_wx_user is
'微信开放接口用户信息';

comment on column t_wx_user.id is
'编号';

comment on column t_wx_user.Subscribe is
'是否订阅';

comment on column t_wx_user.Subscribe_Time is
'订阅时间';

comment on column t_wx_user.wx_open_id is
'微信公众号openID';

comment on column t_wx_user.mp_open_id is
'微信开放平台openID';

comment on column t_wx_user.pay_open_id is
'用于微信支付的关系公众号openID';

comment on column t_wx_user.union_ID is
'联合ID';

comment on column t_wx_user.Group_ID is
'组编码';

comment on column t_wx_user.open_id is
'openID';

comment on column t_wx_user.Tag_ID_List is
'标签编码组';

comment on column t_wx_user.Nickname is
'昵称';

comment on column t_wx_user.Sex is
'性别';

comment on column t_wx_user.Language is
'语言';

comment on column t_wx_user.City is
'城市';

comment on column t_wx_user.Province is
'省份';

comment on column t_wx_user.Country is
'国家';

comment on column t_wx_user.Head_img_URL is
'头像';

comment on column t_wx_user.Privilege is
'权限';

comment on column t_wx_user.QR_Scene is
'二维码';

comment on column t_wx_user.Subscribe_Scene is
'订阅场景';

comment on column t_wx_user.QR_Scene_Str is
'一维码';

comment on column t_wx_user.Err_Code is
'错误编码';

comment on column t_wx_user.Err_Msg is
'错误信息';

comment on column t_wx_user.creator is
'本数据创建者';

comment on column t_wx_user.create_time is
'生成时间';

comment on column t_wx_user.updated_by is
'更新者';

comment on column t_wx_user.update_time is
'帐号信息更新时间';

comment on column t_wx_user.domain_id is
'数据隶属';

comment on column t_wx_user.Remark is
'备注';

comment on column t_wx_user.addi is
'附加信息';

comment on column t_wx_user.status is
'状态,00: 有效, 02: 禁止登录, 04: 锁定, 06: 攻击者, 08: 过期';

/*==============================================================*/
/* Index: idx_wx_user_full                                      */
/*==============================================================*/
create  index if not exists  idx_wx_user_full on t_wx_user (
wx_open_id,
mp_open_id,
union_ID,
Group_ID,
Nickname
);

/*==============================================================*/
/* Index: idx_wx_user_openid                                    */
/*==============================================================*/
create unique index if not exists  idx_wx_user_openid on t_wx_user (
union_ID
);

/*==============================================================*/
/* Table: t_xkb_user                                            */
/*==============================================================*/
create table if not exists  t_xkb_user (
   id                   INT8                 not null,
   school_id            INT8                 null,
   subdistrict          VARCHAR              null,
   faculty              VARCHAR              null,
   grade                VARCHAR              null,
   class                VARCHAR              null,
   domain_id            INT8                 null,
   creator              INT8                 null,
   create_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   updated_by           INT8                 null,
   update_time          INT8                 null default (extract(epoch from current_timestamp)*1000)::bigint,
   addi                 JSONB                null,
   remark               VARCHAR              null,
   status               VARCHAR              null default '01',
   constraint PK_T_XKB_USER primary key (id)
);

comment on table t_xkb_user is
'校快保补充用户信息';

comment on column t_xkb_user.id is
'编号';

comment on column t_xkb_user.school_id is
'学校';

comment on column t_xkb_user.subdistrict is
'校区';

comment on column t_xkb_user.faculty is
'学院';

comment on column t_xkb_user.grade is
'年级';

comment on column t_xkb_user.class is
'班级';

comment on column t_xkb_user.domain_id is
'数据属主';

comment on column t_xkb_user.creator is
'创建者用户ID';

comment on column t_xkb_user.create_time is
'创建时间';

comment on column t_xkb_user.updated_by is
'更新者';

comment on column t_xkb_user.update_time is
'修改时间';

comment on column t_xkb_user.addi is
'附加信息';

comment on column t_xkb_user.remark is
'备注';

comment on column t_xkb_user.status is
'状态，00：草稿，01：有效，02：作废';

-- ALTER SEQUENCE t_xkb_user_id_seq RESTART WITH 20000;


/*==============================================================*/
/* View: v_aa                                                   */
/*==============================================================*/
create or replace view v_aa as
select 

    da.id as domain_api_id,
    da.data_scope->>'data' as da_grant_data,
    da.data_scope->>'type' as da_grant_type,
    da.grant_source da_grant_source,
    da.data_access_mode da_data_access_mode,


    ud.id as user_domain_id,
    ud.data_scope->>'data' as ud_grant_data,
    ud.data_scope->>'type' as ud_grant_type,
    ud.grant_source ud_grant_source,
    ud.data_access_mode ud_data_access_mode,

    u.id as user_id,
    coalesce(u.official_name,u.nickname,u.mobile_phone,u.account,u.id::text) as user_name,
    u.mobile_phone as mobile_phone,

    d.name as domain_name,
    d.id as domain_id,
    d.domain,d.priority,
    a.id as api_id,
    a.name as api_name,
    a.expose_path as api
from t_domain_api da
left join t_user_domain ud on ud.domain=da.domain
left join t_user u on ud.sys_user = u.id
left join t_domain d on d.id = da.domain
left join t_api a on a.id =da.api;

comment on view v_aa is
'authorization
authentication';

drop table if exists t_v_aa;
create table t_v_aa as select * from v_aa;

/*==============================================================*/
/* View: v_api_domain                                           */
/*==============================================================*/
create or replace view v_api_domain as
select a.id,
a.id as api_id,a.name as api_name,a.expose_path,
d.id as domain_id, d.name as domain_name,d.priority,
da.grant_source,da.data_access_mode,da.data_scope
from t_api a
left join t_domain_api da on da.api=a.id
left join t_domain d on da.domain=d.id
order by a.id,da.id;

comment on view v_api_domain is
'api domain';

drop table if exists t_v_api_domain;
create table if not exists t_v_api_domain as select * from v_api_domain;

/*==============================================================*/
/* View: v_authenticate                                         */
/*==============================================================*/
create or replace view v_authenticate as
select 
    da.id as grant_id,
    da.data_scope->>'data' as grant_data,
    da.data_scope->>'type' as grant_type,
    da.grant_source,
    da.data_access_mode,

    u.id as user_id,
    coalesce(u.official_name,u.nickname,u.mobile_phone,u.account,u.id::text) as user_name,
    u.mobile_phone as mobile_phone,

    d.name as domain_name,
    d.id as domain_id,
    d.domain,d.priority,
    a.id as api_id,
    a.name as api_name,
    a.expose_path as api
from t_domain_api da
left join t_user_domain ud on ud.domain=da.domain
left join t_user u on ud.sys_user = u.id
left join t_domain d on d.id = da.domain
left join t_api a on a.id =da.api;

comment on view v_authenticate is
'v_authenticate';

drop table if exists t_v_authenticate;
create table t_v_authenticate as select * from v_authenticate;

/*==============================================================*/
/* View: v_domain_api                                           */
/*==============================================================*/
create or replace view v_domain_api as
select 
    da.id,
    d.id as auth_domain_id,
    d.name as domain_name,
    d.domain,d.priority,
    a.id as api_id,
    a.name as api_name,
    a.expose_path,
    a.access_control_level,
    da.domain_id, 
    da.grant_source,
    da.data_access_mode,
    da.data_scope,da.create_time,da.remark,da.addi,da.creator,da.status
from t_domain_api da
	join t_domain d on da.domain=d.id
	join t_api a on da.api=a.id
	order by d.id,a.id;

comment on view v_domain_api is
'域,API';

drop table if exists t_v_domain_api;
create table if not exists t_v_domain_api as select * from v_domain_api limit 1;

/*==============================================================*/
/* View: v_domain_asset                                         */
/*==============================================================*/
create or replace view v_domain_asset as
select 
  d.id,d.name domain_name,d.domain,d.priority,
  da.id domain_asset_id,
  u.id user_id,account,official_name,id_card_type,id_card_no,email,nickname,mobile_phone,u.status user_status,
  coalesce(official_name,nickname,mobile_phone,account,u.id::text) as user_name,
  a.id api_id,a.name api_name,expose_path
from t_domain d
  left join t_domain_asset da on  da.domain_id=d.id
  left join t_api a on r_type='da' and a.id=asset_id
  left join t_user u on r_type='ud' and u.id=asset_id
order by d.domain,a.expose_path,u.id;

comment on view v_domain_asset is
'domain user relation
domain api relation';

drop table if exists t_v_domain_asset;
create table if not exists t_v_domain_asset as select * from v_domain_asset limit 1;

/*==============================================================*/
/* View: v_domain_user                                          */
/*==============================================================*/
create or replace view v_domain_user as
select d.id,
  d.id as domain_id,d.name as domain_name,d.priority,
  u.id as user_id,
  coalesce(
    u.official_name, 
    u.nickname, 
    u.mobile_phone, 
    u.account, 
    u.id::text::character varying) AS user_name
from t_domain d
  left join t_user_domain ud on d.id=ud.domain
  left join t_user u on ud.sys_user=u.id
  order by d.id,u.id;

comment on view v_domain_user is
'域用户视图';

drop table if exists t_v_domain_user;
create table if not exists t_v_domain_user as select * from v_domain_user;

/*==============================================================*/
/* View: v_exam_file                                            */
/*==============================================================*/
create or replace view v_exam_file as
 SELECT ei.id AS exam_id,
    f.id AS file_id,
    f.digest,
    f.file_name,
    f.size,
    f.path,
    f.domain_id AS file_domain_id,
    f.creator AS file_creator,
    ei.creator AS exam_creator
   FROM t_exam_info ei
     CROSS JOIN LATERAL jsonb_array_elements_text(ei.files) file_id_text(value)
     JOIN t_file f ON f.id = file_id_text.value::bigint
  WHERE ei.files IS NOT NULL AND jsonb_typeof(ei.files) = 'array'::text AND ei.status::text != '12'::text AND f.status::text != '02'::text;

comment on view v_exam_file is
'v_exam_file';

drop table if exists t_v_exam_file;

create table t_v_exam_file as select * from v_exam_file;

/*==============================================================*/
/* View: v_exam_paper                                           */
/*==============================================================*/
create or replace view v_exam_paper as
WITH
question_agg AS(
	SELECT 
    	group_id,-- 组合数据时，根据group_id找到对应的题目数组
    	jsonb_agg(
        	jsonb_build_object(
            	'id', id,
                'type', type,
                'content', content,
                'options', options,
                'answers', answers,
                'score', score, 
                'analysis', analysis,
                'title', title,
                'answer_file_path', answer_file_path,
                'test_file_path', test_file_path,
                'input', input,
                'output', output,
                'example', example,
                'repo', repo,
                'order', "order", 
                'group_id', group_id,
                'status', status,
                'question_attachments_path', question_attachments_path
            ) ORDER BY "order" -- 在构成json，就是插入数组的过程，就需要顺序判别
        )AS questions ,-- 起别名，能取出数组
        SUM(score) AS group_total_score,
        COUNT(id) AS group_question_count
    FROM t_exam_paper_question
    WHERE status = '00'
    GROUP BY group_id
),
-- 构建题组数据
group_data AS (
	SELECT 
    	pg.id,
        pg.name,
        pg."order",
        pg.creator,
        pg.create_time,
        pg.updated_by,
        pg.update_time,
        pg.status,
        pg.addi,
        pg.exam_paper_id,
        COALESCE(qa.questions, '[]'::jsonb) AS questions, -- 取出前面阶段构建好的题目
        COALESCE(qa.group_total_score, 0) AS group_total_score,
        COALESCE(qa.group_question_count, 0) AS group_question_count
    FROM t_exam_paper_group pg
    LEFT JOIN question_agg qa ON qa.group_id = pg.id
    WHERE pg.status != '02'
),

-- 聚合题组数据为json
paper_groups AS (
	SELECT 
    	exam_paper_id,
    	jsonb_agg(
        	jsonb_build_object(
            	'id',id,
    			'name',name,
                'order',"order",
                'creator',creator,
                'create_time',create_time,
                'updated_by',updated_by,
                'update_time',update_time,
                'status',status,
                'addi',addi,
                'questions',questions
            ) ORDER BY "order"
        ) AS groups_data,
        SUM(group_total_score) AS total_score,
        SUM(group_question_count) AS question_count,
        COUNT(*) AS group_count
    FROM group_data
    GROUP BY exam_paper_id
)
SELECT 	
	p.id,
	p.exam_session_id,
    p.practice_id,
	p.name,
	p.creator,
	p.create_time,
	p.updated_by,
	p.update_time,
	p.status,
    COALESCE(pgrp.total_score, 0) AS total_score,
    COALESCE(pgrp.question_count, 0) AS question_count,
    COALESCE(pgrp.group_count, 0) AS group_count,
    COALESCE(pgrp.groups_data, '[]'::jsonb) AS groups_data
FROM t_exam_paper p 
LEFT JOIN paper_groups pgrp ON pgrp.exam_paper_id = p.id
WHERE p.status = '00';

comment on view v_exam_paper is
'考卷';

drop table if exists t_v_exam_paper;

create table t_v_exam_paper as select * from v_exam_paper;

/*==============================================================*/
/* View: v_exam_respondent_count                                */
/*==============================================================*/
create or replace view v_exam_respondent_count as
SELECT es.id                                     AS exam_session_id,
       COALESCE(count(DISTINCT e.id), 0::bigint) AS respondent_count
FROM t_exam_session es
         LEFT JOIN t_examinee e ON es.id = e.exam_session_id AND e.status::text = '10'::text
GROUP BY es.id;

comment on view v_exam_respondent_count is
'考试作答人数统计视图';

drop table if exists t_v_exam_respondent_count;

create table t_v_exam_respondent_count as select * from v_exam_respondent_count;

/*==============================================================*/
/* View: v_exam_teacher_marked_count                            */
/*==============================================================*/
create or replace view v_exam_teacher_marked_count as
 SELECT e.exam_session_id,
    m.teacher_id,
    COALESCE(count(DISTINCT e.id), 0::bigint) AS marked_count
   FROM t_mark m
     LEFT JOIN t_examinee e ON m.examinee_id = e.id AND e.status::text = '10'::text
  WHERE m.score IS NOT NULL
  GROUP BY e.exam_session_id, m.teacher_id;

comment on view v_exam_teacher_marked_count is
'某场考试教师已批改的学生人数统计视图';

drop table if exists t_v_exam_teacher_marked_count;

create table t_v_exam_teacher_marked_count as select * from v_exam_teacher_marked_count;

/*==============================================================*/
/* View: v_exam_unmarked_student_count                          */
/*==============================================================*/
create or replace view v_exam_unmarked_student_count as
SELECT e.exam_session_id,
       COALESCE(count(DISTINCT sa.examinee_id), 0::bigint) AS unmarked_student_count
FROM t_examinee e
         LEFT JOIN t_student_answers sa ON sa.examinee_id = e.id AND sa.answer_score IS NULL
WHERE e.status::text = '10'::text
GROUP BY e.exam_session_id;

comment on view v_exam_unmarked_student_count is
'考试待批改人数统计视图';

drop table if exists t_v_exam_unmarked_student_count;

create table t_v_exam_unmarked_student_count as select * from v_exam_unmarked_student_count;

/*==============================================================*/
/* View: v_examinee_info                                        */
/*==============================================================*/
create or replace view v_examinee_info as
 SELECT DISTINCT examinees.id,
    examinees.student_id,
    users.account,
    users.mobile_phone,
    users.user_token,
    users.official_name,
    users.id_card_no,
    examinees.examinee_number,
    exam_infos.id AS exam_id,
    exam_infos.name AS exam_name,
    exam_sessions.id AS exam_session_id,
    exam_papers.id AS exam_paper_id,
    exam_papers.name AS exam_paper_name,
    examinees.exam_room AS exam_room_id,
    exam_rooms.name AS exam_room_name,
    examinees.extra_time,
    COALESCE(next_sessions.start_time - (exam_sessions.end_time + examinees.extra_time), (24 * 60 * 60 * 1000)::bigint) AS extendable_time,
    exam_sessions.start_time,
    exam_sessions.end_time,
    CASE 
        WHEN exam_sessions.period_mode = '02' AND examinees.start_time IS NOT NULL 
            THEN examinees.start_time + (exam_sessions.duration * 60 * 1000)
        ELSE COALESCE(exam_sessions.end_time + examinees.extra_time, exam_sessions.end_time)
    END AS actual_end_time,
    examinees.status AS examinee_status,
    examinees.remark,
    exam_sessions.period_mode,
    COALESCE(exam_sessions.start_time + exam_sessions.late_entry_time * 60 * 1000, exam_sessions.start_time) AS allow_entry_time,
    COALESCE(exam_sessions.end_time - exam_sessions.early_submission_time * 60 * 1000, exam_sessions.end_time) AS allow_submit_time,
    exam_infos.mode,
    examinees.end_time AS examinee_end_time,
    examinees.start_time AS examinee_start_time,
    examinees.serial_number
   FROM t_examinee examinees
     JOIN t_exam_session exam_sessions ON exam_sessions.id = examinees.exam_session_id
     JOIN t_exam_info exam_infos ON exam_infos.id = exam_sessions.exam_id
     JOIN t_exam_paper exam_papers ON exam_papers.exam_session_id = exam_sessions.id
     LEFT JOIN t_exam_room exam_rooms ON exam_rooms.id = examinees.exam_room
     JOIN t_user users ON users.id = examinees.student_id
     LEFT JOIN LATERAL ( SELECT exam_sessions_1.start_time
           FROM t_exam_session exam_sessions_1
          WHERE exam_sessions_1.exam_id = exam_infos.id AND exam_sessions_1.start_time > (exam_sessions.end_time + examinees.extra_time)
          ORDER BY exam_sessions_1.start_time
         LIMIT 1) next_sessions ON true
  GROUP BY examinees.id, exam_sessions.id, exam_infos.id, exam_papers.id, exam_rooms.id, users.id, exam_sessions.start_time, exam_sessions.end_time, examinees.extra_time, next_sessions.start_time
  ORDER BY examinees.id DESC;

comment on view v_examinee_info is
'考生视图';

drop table if exists t_v_examinee_info;

create table t_v_examinee_info as select * from v_examinee_info;

/*==============================================================*/
/* View: v_insurance_policy                                     */
/*==============================================================*/
create or replace view v_insurance_policy as
select
    i.id,
    i.order_id,
    i.sn,
    i.name,
    i.policy,
    i.start,
    i.cease,
    i.year,
    i.duration,
    i.premium,
    i.policy_scheme,
    i.create_time,
    i.update_time,
    i.sn_creator,
    i.creator,
    i.addi,
    i.remark,
    i.status,
    i.insurance_type_id,
    i.is_entry_policy,
    i.favorite,
    o.trade_no,
    o.pay_order_no,
    o.insure_order_no,
    o.Org_ID,
    o.plan_id,
    o.batch,
    o.Insured_id,
    o.Policyholder_id,
    o.Create_Time order_create_time,
    o.Pay_Time,
    o.Pay_Type,
    o.amount,
    o.unit_price,
    o.commence_date,
    o.expiry_date,
    o.indate,
    o.charge_mode,
    o.Relation,
    o.Insurance_Type,
    o.policy_doc,
    o.Same,
    o.Status order_status,
    o.creator order_creator,
    o.insurer,
    u.Official_Name I_Official_Name,
    u.ID_Card_Type I_ID_Card_Type,
    u.ID_Card_No I_ID_Card_No ,
    u.Mobile_Phone I_Mobile_Phone,
    u.Gender I_Gender,
    u.Birthday I_Birthday,
    u.Addi I_Addi,  
    h.Official_Name h_Official_Name,
    h.ID_Card_Type h_ID_Card_Type,
    h.ID_Card_No h_ID_Card_No,
    h.Mobile_Phone h_Mobile_Phone,
    h.Addi h_Addi,
    x.Subdistrict,
    x.Faculty,
    x.Grade,
    x.Class,
    x.Create_Time x_Create_Time,
    s.Name school,
    s.Faculty s_Faculty,
    s.Branches s_Branches,
    s.Category s_Category,
    s.Province,
    s.City,
    s.District,
    s.Data_Sync_Target,
    s.Sale_Managers,
    s.School_Managers,
    s.purchase_rule,
    s.Create_Time s_Create_Time,
    
    it.parent_id AS insurance_type_parent_id,
    
    a.id insure_attach_id,
    a.policy_no, 
    a.others, 
    a.files, 
    a.addi attach_addi, 
    a.create_time attach_create_time, 
    a.update_time  attach_update_time, 
    a.creator  attach_creator,
    (SELECT
			CASE
				WHEN a.files @> '[{"label":"保单"}]' THEN '0'
			ELSE '2'
			END) AS policy_upload_status,
    (SELECT
			CASE
			WHEN o.files @> '[{"label":"发票"}]' THEN '0'
			ELSE '2'
			END) AS invoice_upload_status
from t_insurance_policy i 
	left join t_order o on i.order_id=o.id
	left join t_school s on o.org_id= s.id 
	left join t_user u on o.Insured_id=u.id
	left join t_user h on o.Policyholder_id=h.id
	left join t_xkb_user x on o.Insured_id=x.id
  left join t_insurance_types it on i.insurance_type_id = it.id
  left join t_insure_attach a on i.id = a.insure_policy_id
where i.insurance_type_id = 10040 and o.charge_mode= '8'
union all
select
    i.id,
    i.order_id,
    i.sn,
    i.name,
    i.policy,
    i.start,
    i.cease,
    i.year,
    i.duration,
    i.premium,
    i.policy_scheme,
    i.create_time,
    i.update_time,
    i.sn_creator,
    i.creator,
    i.addi,
    i.remark,
    i.status,
    i.insurance_type_id,
    i.is_entry_policy,
    i.favorite,
    o.trade_no,
    o.pay_order_no,
    o.insure_order_no,
    o.Org_ID,
    o.plan_id,
    o.batch,
    o.Insured_id,
    o.Policyholder_id,
    o.Create_Time order_create_time,
    o.Pay_Time,
    o.Pay_Type,
    o.amount,
    o.unit_price,
    o.commence_date,
    o.expiry_date,
    o.indate,
    o.charge_mode,
    o.Relation,
    o.Insurance_Type,
    o.policy_doc,
    o.Same,
    o.Status order_status,
    o.creator order_creator,
    o.insurer,
    u.Official_Name I_Official_Name,
    u.ID_Card_Type I_ID_Card_Type,
    u.ID_Card_No I_ID_Card_No ,
    u.Mobile_Phone I_Mobile_Phone,
    u.Gender I_Gender,
    u.Birthday I_Birthday,
    u.Addi I_Addi,  
    h.Official_Name h_Official_Name,
    h.ID_Card_Type h_ID_Card_Type,
    h.ID_Card_No h_ID_Card_No,
    h.Mobile_Phone h_Mobile_Phone,
    h.Addi h_Addi,
    x.Subdistrict,
    x.Faculty,
    x.Grade,
    x.Class,
    x.Create_Time x_Create_Time,
    s.Name school,
    s.Faculty s_Faculty,
    s.Branches s_Branches,
    s.Category s_Category,
    s.Province,
    s.City,
    s.District,
    s.Data_Sync_Target,
    s.Sale_Managers,
    s.School_Managers,
    s.purchase_rule,
    s.Create_Time s_Create_Time,
    
    it.parent_id AS insurance_type_parent_id,
    
    a.id insure_attach_id,
    a.policy_no, 
    a.others, 
    a.files, 
    a.addi attach_addi, 
    a.create_time attach_create_time, 
    a.update_time  attach_update_time, 
    a.creator  attach_creator,
    (SELECT
			CASE
				WHEN a.files @> '[{"label":"保单"}]' THEN '0'
			ELSE '2'
			END) AS policy_upload_status,
		(SELECT
			CASE
				WHEN o.files @> '[{"label":"发票"}]' THEN '0'
			ELSE '2'
			END) AS invoice_upload_status
from t_insurance_policy i 
	left join t_order o on i.order_id=o.id
	left join t_school s on o.org_id= s.id 
	left join t_user u on o.Insured_id=u.id
	left join t_user h on o.Policyholder_id=h.id
	left join t_xkb_user x on o.Insured_id=x.id
  left join t_insurance_types it on i.insurance_type_id = it.id
  left join t_insure_attach a on 
				 o.org_id = a.school_id 
				 and o.batch=a.batch 
				 and x.grade=a.grade 
				 and a.year=i.year
where i.insurance_type_id != 10040 or o.charge_mode != '8';

comment on view v_insurance_policy is
'保单全视图';

drop table if exists t_v_insurance_policy;
create table t_v_insurance_policy as select * from v_insurance_policy limit 1;

/*==============================================================*/
/* View: v_insurance_policy2                                    */
/*==============================================================*/
create or replace view v_insurance_policy2 as
SELECT i.id,
    i.sn,
    i.sn_creator,
    i.name,
    i.order_id,
    i.policy,
    i.start,
    i.cease,
    i.year,
    i.duration,
    i.premium,
    i.third_party_premium,
    i.insurance_type_id,
    o.plan_id,
    i.create_time,
    i.update_time,
    i.pay_time,
    i.pay_channel,
    i.pay_type,
    i.unit_price,
    i.external_status,
    o.org_id,
    o.insurer,
    i.org_manager_id,
    i.insurance_type,
    i.policy_scheme,
    i.activity_name,
    i.activity_category,
    i.activity_desc,
    i.activity_location,
    i.activity_date_set,
    i.insured_count,
    i.compulsory_student_num,
    i.non_compulsory_student_num,
    i.contact,
    i.fee_scheme,
    i.car_service_target,
    i.policy_enroll_time,
    i.policyholder,
    i.policyholder_type,
    i.policyholder_id,
    i.policyholder ->> 'Name'::text AS org_name,
    i.policyholder ->> 'Province'::text AS org_province,
    i.policyholder ->> 'City'::text AS org_city,
    i.policyholder ->> 'District'::text AS org_district,
    i.policyholder ->> 'SchoolCategory'::text AS org_school_category,
    i.policyholder ->> 'IsCompulsory'::text AS org_is_compulsory,
    i.policyholder ->> 'IsSchool'::text AS org_is_school,
    i.insured ->> 'Province'::text AS insured_province,
    i.insured ->> 'City'::text AS insured_city,
    i.insured ->> 'District'::text AS insured_district,
    i.insured ->> 'SchoolCategory'::text AS insured_school_category,
    (i.insured ->> 'IsCompulsory'::text)::boolean AS insured_is_compulsory,
    i.insured ->> 'Name'::text AS insured_name,
    i.insured ->> 'Category'::text AS insured_category,
    (select  sum((list ->'DriverSeatNumber')::int)  driver_seat_sum
    from jsonb_array_elements(i.insured_list) as list where i.insured_list <> '{}' and i.insurance_type_id = 10028) ,
    (select sum((list ->'SeatNum')::int)  seat_sum
    from jsonb_array_elements(i.insured_list) as list where i.insured_list <> '{}' and i.insurance_type_id = 10028) ,
    i.same,
    i.relation,
    i.insured,
    i.insured_id,
    i.have_insured_list,
    i.insured_group_by_day,
    i.insured_type,
    i.insured_list,
    i.indate,
    i.jurisdiction,
    i.dispute_handling,
    i.prev_policy_no,
    i.insure_base,
    i.blanket_insure_code,
    i.custom_type,
    i.train_projects,
    i.business_locations,
    i.pool_num,
  
    i.have_dinner_num,
    i.open_pool_num,
    i.heated_pool_num,
    i.training_pool_num,
    i.inner_area,
    i.outer_area,
    i.pool_name,
    i.arbitral_agency,   
    
    i.dinner_num,
    i.canteen_num,
    i.shop_num,
    i.have_rides,
    i.have_explosive,
    i.area,
    i.traffic_num,
    i.temperature_type,
    i.is_indoor,
    i.extra,
    i.bank_account,
    i.pay_contact,
    i.sudden_death_terms,
    i.have_sudden_death,
    i.spec_agreement,
    i.third_party_account,
    i.creator,
    i.domain_id,
    i.status,
    i.is_entry_policy,
    i.is_admin_pay,
    i.favorite,
    i.cancel_desc,
    i.zero_pay_status,
    a.others,
    a.files,
    a.insure_policy_id,
    a.status AS a_status,
    ( SELECT
				CASE
					WHEN EXTRACT(EPOCH FROM now())::bigint*1000 > i.cease THEN ( SELECT '已过期')
					WHEN EXTRACT(EPOCH FROM now())::bigint*1000 < i.start THEN ( SELECT '未起保')
					ELSE ( SELECT '保障中')
				END) AS policy_status,
								
    it.parent_id AS insurance_type_parent_id,
    ( SELECT
                CASE
                    WHEN it.parent_id = 10000 THEN ( SELECT it2.name
                       FROM t_insurance_types it2
                      WHERE it2.id = i.insurance_type_id)
                    WHEN it.parent_id = 10040 THEN ( SELECT it2.name
                       FROM t_insurance_types it2
                      WHERE it2.id = 10040)
                    ELSE ( SELECT o.insurance_type)
                END AS insurance_type) AS insurance_display,
    o.charge_mode,
    o.insurance_company,
    o.insurance_company_account,
    o.create_time AS order_create_time,
    o.is_invoice,
    o.inv_borrow,
    o.inv_visible,
    o.inv_title,
    o.inv_status,
    o.files AS o_files,
    ( SELECT
                CASE
                    WHEN a.files @> '[{"label":"保单"}]' THEN '0'
                    ELSE '2'
                END) AS policy_upload_status,
    ( SELECT
                    CASE
                    WHEN o.files @> '[{"label":"发票"}]' THEN '0'
                            ELSE '2'
                END) AS invoice_upload_status
from t_insurance_policy i
     LEFT JOIN t_insurance_types it ON i.insurance_type_id = it.id
     LEFT JOIN t_order o ON i.order_id = o.id
     LEFT JOIN t_xkb_user x ON o.insured_id = x.id
     LEFT JOIN t_insure_attach a ON
        o.org_id = a.school_id 
				AND o.batch = a.batch
				AND x.grade = a.grade
				AND a.year = i.year
where 
	i.insurance_type_id = 10040 and o.charge_mode <> '8' 
	
union all
SELECT i.id,
    i.sn,
    i.sn_creator,
    i.name,
    i.order_id,
    i.policy,
    i.start,
    i.cease,
    i.year,
    i.duration,
    i.premium,
    i.third_party_premium,
    i.insurance_type_id,
    o.plan_id,
    i.create_time,
    i.update_time,
    i.pay_time,
    i.pay_channel,
    i.pay_type,
    i.unit_price,
    i.external_status,
    o.org_id,
    o.insurer,
    i.org_manager_id,
    i.insurance_type,
    i.policy_scheme,
    i.activity_name,
    i.activity_category,
    i.activity_desc,
    i.activity_location,
    i.activity_date_set,
    i.insured_count,
    i.compulsory_student_num,
    i.non_compulsory_student_num,
    i.contact,
    i.fee_scheme,
    i.car_service_target,
    i.policy_enroll_time,
    i.policyholder,
    i.policyholder_type,
    i.policyholder_id,
    i.policyholder ->> 'Name'::text AS org_name,
    i.policyholder ->> 'Province'::text AS org_province,
    i.policyholder ->> 'City'::text AS org_city,
    i.policyholder ->> 'District'::text AS org_district,
    i.policyholder ->> 'SchoolCategory'::text AS org_school_category,
    i.policyholder ->> 'IsCompulsory'::text AS org_is_compulsory,
    i.policyholder ->> 'IsSchool'::text AS org_is_school,
    i.insured ->> 'Province'::text AS insured_province,
    i.insured ->> 'City'::text AS insured_city,
    i.insured ->> 'District'::text AS insured_district,
    i.insured ->> 'SchoolCategory'::text AS insured_school_category,
    (i.insured ->> 'IsCompulsory'::text)::boolean AS insured_is_compulsory,
    i.insured ->> 'Name'::text AS insured_name,
    i.insured ->> 'Category'::text AS insured_category,
    (select  sum((list ->'DriverSeatNumber')::int)  driver_seat_sum
    from jsonb_array_elements(i.insured_list) as list where i.insured_list <> '{}' and i.insurance_type_id = 10028) ,
    (select sum((list ->'SeatNum')::int)  seat_sum
    from jsonb_array_elements(i.insured_list) as list where i.insured_list <> '{}' and i.insurance_type_id = 10028) ,
    i.same,
    i.relation,
    i.insured,
    i.insured_id,
    i.have_insured_list,
    i.insured_group_by_day,
    i.insured_type,
    i.insured_list,
    i.indate,
    i.jurisdiction,
    i.dispute_handling,
    i.prev_policy_no,
    i.insure_base,
    i.blanket_insure_code,
    i.custom_type,
    i.train_projects,
    i.business_locations,
    i.pool_num,
  
    i.have_dinner_num,
    i.open_pool_num,
    i.heated_pool_num,
    i.training_pool_num,
    i.inner_area,
    i.outer_area,
    i.pool_name,
    i.arbitral_agency,   
    
    i.dinner_num,
    i.canteen_num,
    i.shop_num,
    i.have_rides,
    i.have_explosive,
    i.area,
    i.traffic_num,
    i.temperature_type,
    i.is_indoor,
    i.extra,
    i.bank_account,
    i.pay_contact,
    i.sudden_death_terms,
    i.have_sudden_death,
    i.spec_agreement,
    i.third_party_account,
    i.creator,
    i.domain_id,
    i.status,
    i.is_entry_policy,
    i.is_admin_pay,
    i.favorite,
    i.cancel_desc,
    i.zero_pay_status,
    a.others,
    a.files,
    a.insure_policy_id,
    a.status AS a_status,
    ( SELECT
				CASE
					WHEN EXTRACT(EPOCH FROM now())::bigint*1000 > i.cease THEN ( SELECT '已过期')
					WHEN EXTRACT(EPOCH FROM now())::bigint*1000 < i.start THEN ( SELECT '未起保')
				ELSE ( SELECT '保障中')
				END) AS policy_status,
    it.parent_id AS insurance_type_parent_id,
    ( SELECT
				CASE
					WHEN it.parent_id = 10000 THEN ( SELECT it2.name
						 FROM t_insurance_types it2
						WHERE it2.id = i.insurance_type_id)
					WHEN it.parent_id = 10040 THEN ( SELECT it2.name
						 FROM t_insurance_types it2
						WHERE it2.id = 10040)
					ELSE ( SELECT o.insurance_type)
				END AS insurance_type) AS insurance_display,
    o.charge_mode,
    o.insurance_company,
    o.insurance_company_account,
    o.create_time AS order_create_time,
    o.is_invoice,
    o.inv_borrow,
    o.inv_visible,
    o.inv_title,
    o.inv_status,
    o.files AS o_files,
    (SELECT
				CASE
						WHEN a.files @> '[{"label":"保单"}]' THEN '0'
				ELSE '2'
				END) AS policy_upload_status,
    (SELECT
				CASE
					WHEN o.files @> '[{"label":"发票"}]' THEN '0'
				ELSE '2'
				END) AS invoice_upload_status
from t_insurance_policy i
     LEFT JOIN t_insurance_types it ON i.insurance_type_id = it.id
     LEFT JOIN t_order o ON i.order_id = o.id
     LEFT JOIN t_xkb_user x ON o.insured_id = x.id
     LEFT JOIN t_insure_attach a ON i.id = a.insure_policy_id
where
  i.insurance_type_id != 10040 or o.charge_mode = '8';

comment on view v_insurance_policy2 is
'v_insurance_policy2';

drop table if exists t_v_insurance_policy2;
create table t_v_insurance_policy2 as select * from v_insurance_policy2 limit 1;

/*==============================================================*/
/* View: v_insurance_type                                       */
/*==============================================================*/
create or replace view v_insurance_type as
select 	
	i.id,
	i.parent_id,
    i.data_type,

	coalesce(p.name,'险种') as parent_name,
	coalesce(s.name,'险种配置') as org_name,

	coalesce(i.org_id,0) as org_id,
	
	i.layout_order,	
	coalesce(i.insurer,r.insurer) as insurer,
	
	i.ref_id,
 	case when i.ref_id is null then i.name else r.name end as name,
	
	i.alias,

	coalesce(i.pay_type,r.pay_type) as pay_type,
	coalesce(i.pay_name,r.pay_name) as pay_name,
	coalesce(i.pay_channel,r.pay_channel) as pay_channel,
 	i.rule_batch,

	coalesce(i.unit_price,r.unit_price) as unit_price,
	coalesce(i.price,r.price) as price,
	coalesce(i.price_config,r.price_config) as price_config,

 	i.allow_start,
 	i.allow_end,
 
	CASE 
			WHEN 
				extract('epoch' from current_timestamp)*1000 < i.allow_start 	
			THEN 
				'未开始'
			WHEN 
				extract('epoch' from current_timestamp)*1000 > i.allow_end 		
			THEN 
				'已过期'
			WHEN 
				-- 险种方案配置，无规则有效时间
				coalesce(i.org_id,0) = 0 
				or i.allow_start is null or i.allow_end is null
				or extract('epoch' from current_timestamp)*1000 between i.allow_start and i.allow_end 
			THEN 
				'使用中'
			ELSE 
				'无效规则配置'
	END as time_status,     
    
 	i.max_insure_in_year,

	-- 起保，止保时间机构指定的优先，方案配置的为次选
	coalesce(i.insured_start_time,r.insured_start_time) as insured_start_time,
	coalesce(i.insured_end_time,r.insured_end_time) as insured_end_time,
	

	coalesce(i.insured_in_month,r.insured_in_month) as insured_in_month,

	coalesce(i.indate_start,r.indate_start) as indate_start,
	coalesce(i.indate_end,r.indate_end) as indate_end,
	
	i.age_limit,
	
	i.bank_account,
	i.bank_account_name,
	i.bank_name,
	i.bank_id,
	i.floor_price,
	i.define_level,
	i.layout_level,
	i.list_tpl,
	i.files,
	i.pic,
	i.sudden_death_description,
	i.description,
	i.auto_fill,
	i.enable_import_list,
	i.have_dinner_num,
	i.invoice_title_update_times,

	coalesce(i.transfer_auth_files, r.transfer_auth_files) as  transfer_auth_files,
	COALESCE(i.receipt_account, r.receipt_account) AS receipt_account,
	COALESCE(i.contact_qr_code, r.contact_qr_code) AS contact_qr_code,
	i.other_files,
	

	i.contact,

	i.underwriter,
	i.remind_days,
	i.mail,
	i.order_repeat_limit,
	i.group_by_max_day,
	i.web_description,
	i.mobile_description,
	i.auto_fill_param,
	i.interval,
 	coalesce(i.addi,r.addi) as addi,
     coalesce(i.resource, r.resource) AS resource,
    i.create_time,
    i.update_time,
    i.status

-- 规则或方案, ”规则“指针对机构配置的投保参数，”方案“指针对险种配置的投保参数
from t_insurance_types i

-- 所属保险产品，如，校方责任险，学生意外伤害险   
left join t_insurance_types p on i.parent_id=p.id

-- 机构使用的投保规则, 如, 回民小学使用了学意险铂金套餐方案
left join t_insurance_types r on i.ref_id=r.id

-- 机构的信息
left join t_school s on i.org_id=s.id

where 
	i.parent_id=0 
    or i.data_type='4' or i.data_type='6' or i.data_type='8'

order by
	i.parent_id,
	i.org_id,
	i.layout_order,
	i.pay_type;

comment on view v_insurance_type is
'险种方案';

drop table if exists t_v_insurance_type;
create table if not exists t_v_insurance_type as select * from v_insurance_type limit 1;

/*==============================================================*/
/* View: v_insure_attach                                        */
/*==============================================================*/
create or replace view v_insure_attach as
select
   insure_attach_id id,
   Org_ID,
   school,
   s_Category category,
   insurer,insurance_type,
   year,
   batch,
   Grade,
   files,
   policy_no,
   others,
   policy_upload_status,
   invoice_upload_status,
   attach_addi addi,
   attach_create_time create_time,
   attach_update_time update_time,
   attach_creator creator
from
   v_insurance_policy
where
   insurance_type_id = 10040
   and charge_mode != '8'
group by
   insure_attach_id,
   Org_ID,
   school,
   s_category,
   year,
   batch,
   grade,
   files,
   policy_no,
   others,
   insurer,insurance_type,
   policy_upload_status,
   invoice_upload_status,
   attach_addi,
   attach_create_time,
   attach_update_time,
   attach_creator
order by
   school ASC,
   year ASC,
   grade ASC,
   batch ASC;

comment on view v_insure_attach is
'保单附件视图，它实质是保单视图: v_insurance_policy';

drop table if exists t_v_insure_attach;
create table t_v_insure_attach as select * from v_insure_attach limit 1;

/*==============================================================*/
/* View: v_insured_school                                       */
/*==============================================================*/
create or replace view v_insured_school as
select 
s.id,
s.name,
s.category, 
s.province, 
s.city, 
s.district,
s.street,
s.is_school,
s.status org_status,
array_to_json(
 array_agg(
 distinct jsonb_build_object('AllowStart',allow_start,'AllowEnd',allow_end,'CreateTime',i.create_time,'UpdateTime',i.update_time,'RuleSet',i.id_set)
))::jsonb as allow_time
from t_school s left join
 (select allow_start,allow_end,org_id,
  min(create_time) create_time,
  max(update_time) update_time,
  array_agg(id) id_set
  from t_insurance_types 
  where parent_id=10040 group by allow_start,allow_end,org_id 
  order by allow_start) as i 
 on s.id=i.org_id 
 where s.category !='其他' and s.category !='学校'
 group by s.id;

comment on view v_insured_school is
'学生意外险学校列表';

drop table if exists t_v_insured_school;
create table t_v_insured_school as select * from v_insured_school limit 1;

/*==============================================================*/
/* View: v_insurer                                              */
/*==============================================================*/
create or replace view v_insurer as
(with recursive r as (
	select id,name,ref_id,parent_id,insurer 
		from t_insurance_types
		where insurer is not null
	union
	select c.id,c.name,c.ref_id,c.parent_id,coalesce(c.insurer,r.insurer) insurer
		from t_insurance_types c
		join r on c.parent_id=r.id)
	select id,name,ref_id,parent_id,insurer from r order by id)
union
(with recursive r as (
	select id,name,ref_id,parent_id,insurer 
		from t_insurance_types
		where insurer is not null
	union
	select c.id,c.name,c.ref_id,c.parent_id,coalesce(c.insurer,r.insurer) insurer
		from t_insurance_types c
		join r on c.ref_id=r.id)
select id,name,ref_id,parent_id,insurer from r order by id);

comment on view v_insurer is
'查看险种的承保公司，首先取自己的，如果没有，则递归取parent_id指向的险种, 如果还是没有找到，则递归取ref_id指向的险种';

drop table if exists t_v_insurer;
create table t_v_insurer as select * from v_insurer limit 1;

/*==============================================================*/
/* View: v_invigilation_info                                    */
/*==============================================================*/
create or replace view v_invigilation_info as
 WITH invigilation_statistics AS (
         SELECT examinees.exam_session_id,
            examinees.exam_room AS exam_room_id,
            count(DISTINCT examinees.student_id) AS examinee_num,
            sum(
                CASE
                    WHEN examinees.status::text = '02'::text THEN 1
                    ELSE 0
                END) AS absentee_num,
            sum(
                CASE
                    WHEN examinees.status::text = '06'::text THEN 1
                    ELSE 0
                END) AS cheater_num,
            sum(
                CASE
                    WHEN examinees.status::text = '14'::text THEN 1
                    ELSE 0
                END) AS abnormal_examinee_num,
            sum(
                CASE
                    WHEN examinees.extra_time > 0 THEN 1
                    ELSE 0
                END) AS extended_time_num
           FROM t_examinee examinees
          GROUP BY examinees.exam_session_id, examinees.exam_room
        )
 SELECT array_agg(DISTINCT users.id) AS invigilator_ids,
    array_agg(users.official_name) AS invigilator_names,
    count(invigilations.id) AS invigilator_num,
    exam_infos.id AS exam_id,
    exam_infos.type AS exam_type,
    exam_infos.mode AS exam_mode,
    exam_sessions.id AS exam_session_id,
    exam_sessions.start_time,
    exam_sessions.end_time,
    exam_sessions.status,
    exam_sites.id AS exam_site_id,
    exam_sites.name AS exam_site_name,
    exam_rooms.id AS exam_room_id,
    exam_rooms.name AS exam_room_name,
    exam_rooms.capacity AS exam_room_capacity,
    exam_papers.name AS exam_session_name,
    exam_records.content AS record,
    exam_records.basic_eval,
    invi_stat.examinee_num,
    invi_stat.absentee_num,
    invi_stat.cheater_num,
    invi_stat.abnormal_examinee_num,
    invi_stat.extended_time_num
   FROM t_exam_session exam_sessions
     LEFT JOIN t_invigilation invigilations ON invigilations.exam_session_id = exam_sessions.id
     LEFT JOIN t_exam_room exam_rooms ON invigilations.exam_room = exam_rooms.id
     JOIN t_exam_info exam_infos ON exam_infos.id = exam_sessions.exam_id
     LEFT JOIN t_user users ON users.id = invigilations.invigilator
     LEFT JOIN t_exam_paper exam_papers ON exam_papers.exam_session_id = exam_sessions.id
     LEFT JOIN t_exam_site exam_sites ON exam_sites.id = exam_rooms.exam_site
     LEFT JOIN t_exam_record exam_records ON exam_records.exam_session = invigilations.exam_session_id AND exam_records.exam_room = invigilations.exam_room
     LEFT JOIN invigilation_statistics invi_stat ON invi_stat.exam_session_id = exam_sessions.id AND (exam_infos.mode::text = '00'::text OR exam_infos.mode::text = '02'::text AND invi_stat.exam_room_id = exam_rooms.id)
  GROUP BY exam_sessions.id, exam_infos.id, exam_sites.id, exam_rooms.id, exam_papers.name, exam_records.content, exam_records.basic_eval, invi_stat.examinee_num, invi_stat.absentee_num, invi_stat.cheater_num, invi_stat.abnormal_examinee_num, invi_stat.extended_time_num
  ORDER BY exam_sessions.start_time DESC;

comment on view v_invigilation_info is
'监考视图';

drop table if exists t_v_invigilation_info;
create table t_v_invigilation_info as select * from v_invigilation_info;

/*==============================================================*/
/* View: v_latest_pending_mark_practice                         */
/*==============================================================*/
create or replace view v_latest_pending_mark_practice as
 SELECT DISTINCT ON (practice_id, student_id) id AS submission_id,
    practice_id,
    student_id,
    attempt
   FROM t_practice_submissions
  WHERE ((status)::text = '06'::text)
  ORDER BY practice_id, student_id, attempt DESC;

comment on view v_latest_pending_mark_practice is
'v_latest_pending_mark_practice';

 drop table if exists t_v_latest_pending_mark_practice;
create table t_v_latest_pending_mark_practice as select * from v_latest_pending_mark_practice;

/*==============================================================*/
/* View: v_latest_submitted_practice                            */
/*==============================================================*/
create or replace view v_latest_submitted_practice as
 SELECT DISTINCT ON (practice_id, student_id) id AS submission_id,
    practice_id,
    student_id,
    attempt
   FROM t_practice_submissions
  WHERE ((status)::text = '08'::text)
  ORDER BY practice_id, student_id, attempt DESC;

comment on view v_latest_submitted_practice is
'v_latest_submitted_practice';

 drop table if exists t_v_latest_submitted_practice;
create table t_v_latest_submitted_practice as select * from v_latest_submitted_practice;

/*==============================================================*/
/* View: v_latest_unsubmitted_practice                          */
/*==============================================================*/
create or replace view v_latest_unsubmitted_practice as
 SELECT DISTINCT ON (practice_id, student_id) id AS submission_id,
    practice_id,
    student_id,
    attempt
   FROM t_practice_submissions
  WHERE ((status)::text = '00'::text)
  ORDER BY practice_id, student_id, attempt DESC;

comment on view v_latest_unsubmitted_practice is
'v_latest_unsubmitted_practice';

 drop table if exists t_v_latest_unsubmitted_practice;
create table t_v_latest_unsubmitted_practice as select * from v_latest_unsubmitted_practice;

/*==============================================================*/
/* View: v_manager_school                                       */
/*==============================================================*/
create or replace view v_manager_school as
select
left_id as user_ID,
coalesce(u.official_name, u.nickname,u.account,u.id::text) as name,
u.mobile_phone as tel,
right_id as school_ID,
s.name as school_Name,
r.addi->>'role' as user_role,
r.addi->>'type' as rel_type,
r.addi 
from t_relation r
left join t_user u on r.left_id=u.id and left_type='t_user.id'
left join t_school s on r.right_id=s.id and right_type='t_school.id'
order by r.addi->>'type';

comment on view v_manager_school is
'销售/学校管理员/操作员与学校间的关系';

drop table if exists t_v_manager_school;
create table if not exists t_v_manager_school as select * from v_manager_school;

/*==============================================================*/
/* View: v_xkb_school_layout                                    */
/*==============================================================*/
create or replace view v_xkb_school_layout as
select 
  coalesce(s.value,s.name) school,s.id schoolID,s.addi school_addi, 
  coalesce(g.value,g.name) grade,g.id gradeID, g.addi grade_addi,
  coalesce(c.value,c.name) "class",c.id classID, c.addi class_addi
from t_param c 
join t_param g on c.belongto=g.id
join t_param s on g.belongto=s.id
where s.belongto=12000
order by s.id,g.id,c.id;

comment on view v_xkb_school_layout is
'学校/年级/班级配置表';

drop table if exists t_v_xkb_school_layout;
create table t_v_xkb_school_layout as select * from v_xkb_school_layout limit 1;

/*==============================================================*/
/* View: v_xkb_user                                             */
/*==============================================================*/
create or replace view v_xkb_user as
select
   a.id,
   a.account,
   a.official_name,
   a.id_card_type,
   a.id_card_no,
   a.external_id,
   a.external_id_type,
   a.gender,
   a.birthday,
   a.category,
   a.type,
   a.province,
   a.city,
   a.addr,
   a.mobile_phone,
   a.email,
   a.nickname,
   a.avatar,
   a.avatar_type,
   a.role,
   a.grp,
   a.addi,
   a.remark,
   a.status,
   a.create_time,
   a.creator,
   l.school_addi->> 'gradeCount' grade_count,
   l.school_addi->> 'classCount' class_count,
   b.school_id,
   s.purchase_rule,
   s.category school_type,
   s.name school,
   b.subdistrict,
   b.faculty,
   b.grade,
   l.grade_addi->> 'SN' grade_sn,
   b.class,
   l.class_addi->> 'SN' class_sn,
   c.union_id,
   c.wx_open_id,
   c.mp_open_id
from
   t_user a
   left join t_wx_user c on a.id=c.id
   left join t_xkb_user b on  a.id = b.id
   left join t_school s on  b.school_id = s.id
   left join v_xkb_school_layout l on  s.category = l.school and b.grade = l.grade and b.class = l.class;

comment on view v_xkb_user is
'v_xkb_user';

drop table if exists t_v_xkb_user;
create table t_v_xkb_user as select * from v_xkb_user limit 1;

/*==============================================================*/
/* View: v_order                                                */
/*==============================================================*/
create or replace view v_order as
select
  o.ID,
  o.trade_no,
  o.Org_ID,
  o.Insured_id,
  o.Policyholder_id,
  o.sign,
  o.insure_order_no,
  o.pay_order_no,
  o.batch,

  o.Create_Time,
  o.Pay_Time,
  o.Pay_Type,
  o.amount,
  o.unit_price,
  o.commence_date,
  o.expiry_date,
  o.indate,
  o.confirm_refund,
  o.Relation,
  o.Insurance_Type,
  o.plan_id,o.plan_name,o.insurer,
  o.policy_doc,
  o.health_survey,
  o.Same,
  o.files order_files,
  o.addi,
  
  o.Status,
  o.creator,

  u.Official_Name I_Official_Name,
  u.ID_Card_Type I_ID_Card_Type,
  u.ID_Card_No I_ID_Card_No ,
  u.Mobile_Phone I_Mobile_Phone,
  u.Gender I_Gender,
  u.Birthday I_Birthday,
  u.Addi I_Addi,  
 
  h.Official_Name h_Official_Name,
  h.ID_Card_Type h_ID_Card_Type,
  h.ID_Card_No h_ID_Card_No,
  h.Mobile_Phone h_Mobile_Phone,
  h.Addi h_Addi,

  x.Subdistrict,
  x.Faculty,
  x.Grade,
  x.Class,
  x.Create_Time x_Create_Time,

  s.Name school,
  s.Faculty s_Faculty,
  s.Branches s_Branches,
  s.Category s_Category,

  s.Province,
  s.City,
  s.District,
  s.Data_Sync_Target,
  s.Sale_Managers,
  s.School_Managers,
  s.purchase_rule,
  s.Create_Time s_Create_Time
  
 from t_order o
   join t_school s on o.org_id= s.id 
   join t_user u on o.Insured_id=u.id
   join t_user h on o.Policyholder_id=h.id
   join t_xkb_user x on o.Insured_id=x.id;

comment on view v_order is
'订单视图';

drop table if exists t_v_order;
create table t_v_order as select * from v_order limit 1;

/*==============================================================*/
/* View: v_mistake_correct                                      */
/*==============================================================*/
create or replace view v_mistake_correct as
select
m.id,
order_id,

o.org_id,
o.commence_date,
o.expiry_date,
o.insurance_type,

m.official_name,
m.id_card_type,
m.id_card_no,
m.gender,
m.birthday,

x.school,
x.school_id,
x.school_type,

o.insured_id,
x.official_name  original_official_name ,
x.id_card_type	 original_id_card_type,  
x.id_card_no		 original_id_card_no,    
x.gender				 original_gender,        
x.birthday			 original_birthday,      

m.official_name_p,
m.id_card_type_p,
m.id_card_no_p,
m.gender_p,
m.birthday_p,

o.policyholder_id,
u.official_name original_official_name_p ,
u.id_card_type 	original_id_card_type_p,  
u.id_card_no 		original_id_card_no_p,    
u.gender 				original_gender_p,        
u.birthday			original_birthday_p,

m.addi,
m.create_time,
m.update_time,
m.creator,
m.remark,
m.status

from t_mistake_correct m
join t_order o on o.id=m.order_id
left join v_xkb_user x on x.id=o.insured_id
left join t_user u on o.policyholder_id = u.id;

comment on view v_mistake_correct is
'报错视图';

drop table if exists t_v_mistake_correct;
create table t_v_mistake_correct as select * from v_mistake_correct limit 1;




/*==============================================================*/
/* View: v_mistake_correct2                                     */
/*==============================================================*/
create or replace view v_mistake_correct2 as
select
m.id,
m.order_id,
o.insurance_type_id,
it.parent_id AS insurance_type_parent_id,
o.org_id,


o.have_dinner_num,
o.commence_date,
m.commence_date as new_commence_date,
o.expiry_date,
m.expiry_date as new_expiry_date,

o.have_insured_list,
coalesce(modify_type,'2') as modify_type,
o.insurance_type,
o.activity_category,
m.plan_id,
o.plan_id as original_plan_id,
 o.plan_name,
o.policyholder->>'Name' as org_name,
o.policyholder ->> 'Addr' AS org_addr,
o.policyholder ->> 'CreditCode' AS org_credit_code,
o.policyholder ->> 'Contact'  AS org_contact,
o.policyholder ->> 'Phone'  AS org_phone,
o.policyholder ->> 'ContactRole'   AS org_contact_role,
o.policyholder ->> 'CreditCodePic'   AS org_credit_code_pic,
o.policyholder ->> 'SchoolCategory'     AS org_school_category,
 o.policyholder ->> 'Province'   AS org_province,
 o.policyholder ->> 'City'   AS org_city,
 o.policyholder ->> 'District'   AS org_district,
  o.insured ->> 'Name'  AS insured_name,
  o.insured ->> 'Province'  AS insured_province,
  o.insured ->> 'City'   AS insured_city,
  o.insured ->> 'District'    AS insured_district,
 (o.insured ->> 'IsCompulsory')::boolean    AS insured_is_compulsory,
  o.insured ->> 'Category'     AS insured_category,
  o.insured ->> 'SchoolCategory'    AS insured_school_category,


m.official_name,
m.id_card_type,
m.id_card_no,
m.gender,
m.birthday,


x.school,
x.school_id,
x.school_type,


x.official_name  original_official_name ,
x.id_card_type   original_id_card_type,  
x.id_card_no         original_id_card_no,    
x.gender                 original_gender,        
x.birthday           original_birthday,     



m.official_name_p,
m.id_card_type_p,
m.id_card_no_p,
m.gender_p,
m.birthday_p,


u.official_name original_official_name_p ,
u.id_card_type  original_id_card_type_p,  
u.id_card_no        original_id_card_no_p,    
u.gender                original_gender_p,        
u.birthday          original_birthday_p,

o.insurer,
o.activity_name original_activity_name,
m.activity_name,
o.activity_desc original_desc,
m.activity_desc,
o.activity_location original_activity_location,
m.activity_location,
o.activity_date_set original_activity_date_set,
m.activity_date_set activity_date_set,
o.indate original_indate,
m.indate,
o.policyholder original_policyholder,
m.policyholder,
o.policyholder_id,
o.insured original_insured,
m.insured,
o.insured_id,
o.insured_group_by_day original_insured_group_by_day,
m.insured_group_by_day,
o.charge_mode original_charge_mode,
m.charge_mode,
o.amount original_amount,
m.amount,
o.insured_count original_insured_count,
m.insured_count,
o.insured_type original_insured_type,
m.insured_type,
o.insured_list original_insured_list,
m.insured_list,
-- 从m.insured_list中提取insert/update/delete的数据
(select jsonb_agg(list) from jsonb_array_elements(m.insured_list) as list
 where m.insured_list <> '{}' and list->>'Action'='insert') insert_insured_list,
(select jsonb_agg(list) from jsonb_array_elements(m.insured_list) as list
 where m.insured_list <> '{}' and list->>'Action'='delete') delete_insured_list,
(select jsonb_agg(list) from jsonb_array_elements(m.insured_list) as list
 where m.insured_list <> '{}' and list->>'Action'='update') update_insured_list,
--  从原清单中找出会被修改的数据
(select jsonb_agg(a_list) from jsonb_array_elements(o.insured_list) a_list where a_list->>'ID' in
 (select list->>'ID' from jsonb_array_elements(m.insured_list) as list
  where m.insured_list <> '{}' and list->>'Action'='update')) require_update_insured_list,
 o.non_compulsory_student_num original_non_compulsory_student_num,
 m.non_compulsory_student_num,
 o.compulsory_student_num original_compulsory_student_num,
 m.compulsory_student_num,
 o.canteen_num original_canteen_num,
 m.canteen_num,
 o.shop_num original_shop_num,
 m.shop_num,
 o.dinner_num original_dinner_num,
 m.dinner_num,
 o.pay_type original_pay_type,
 m.pay_type,
 o.fee_scheme original_fee_scheme,
 m.fee_scheme,
 m.need_balance,
 o.order_status,
 o.dispute_handling original_dispute_handling,
 m.dispute_handling,
 o.have_sudden_death original_have_sudden_death,
 m.have_sudden_death,
 o.prev_policy_no original_prev_policy_no,
 m.revoked_policy_no,
 m.prev_policy_no,
 o.pool_name original_pool_name,
 m.pool_name,
 o.have_explosive original_have_explosive,
 m.have_explosive,
 o.have_rides original_have_rides,
 m.have_rides,
 o.inner_area original_inner_area,
 m.inner_area,
 o.outer_area original_outer_area,
 m.outer_area,
 o.traffic_num original_traffic_num,
 m.traffic_num,
 o.temperature_type original_temperature_type,
 m.temperature_type,
 o.open_pool_num original_open_pool_num,
 m.open_pool_num,
 o.heated_pool_num original_heated_pool_num,
 m.heated_pool_num,
 o.training_pool_num original_training_pool_num,
 m.training_pool_num,
 o.pool_num original_pool_num,
 m.pool_num,
 o.custom_type original_custom_type,
 m.custom_type,
 o.same original_same,
 m.same,
 o.arbitral_agency original_arbitral_agency,
 m.arbitral_agency,
 m.endorsement_status,
 m.application_files,
 o.balance,
 o.balance_list,
 m.have_negotiated_price,
(select string_agg(p.sn,',') from t_insurance_policy p where p.order_id = m.order_id) as sn,
 
 m.policy_regen,
 m.clear_list,
 m.files_to_remove,
 
 o.policy_scheme original_policy_scheme,
 m.policy_scheme,
 m.invoice_header,
 m.correct_level,
 m.correct_log,
 o.files original_files,
 m.files,
 m.refused_reason,
 m.addi,
 m.create_time,
 m.update_time,
 m.creator,
 m.remark,
 m.status
 from t_mistake_correct m
 join t_order o on o.id=m.order_id
 left join v_xkb_user x on x.id=o.insured_id
 left join t_user u on o.policyholder_id = u.id
 left JOIN t_insurance_types it ON it.id = o.insurance_type_id;

comment on view v_mistake_correct2 is
'报错2';

drop table if exists t_v_mistake_correct2;
create table t_v_mistake_correct2 as select * from v_mistake_correct2 limit 1;

/*==============================================================*/
/* View: v_mistake_correct_show                                 */
/*==============================================================*/
create or replace view v_mistake_correct_show as
select
m.id,
m.order_id,
o.insurance_type_id,
it.parent_id AS insurance_type_parent_id,
o.org_id,

coalesce(m.commence_date,o.commence_date) as commence_date,
coalesce(m.expiry_date,o.expiry_date) as expiry_date,
coalesce(modify_type,'2') as modify_type,
o.have_insured_list,

o.insurance_type,
o.activity_category,
coalesce(m.plan_id,o.plan_id) as plan_id,
coalesce(m.policy_scheme->>'Insurer',o.policy_scheme->>'Insurer') as insurer,
coalesce(m.policy_scheme->>'Name',o.policy_scheme->>'Name') as plan_name,
coalesce(m.policyholder->>'Name',o.policyholder->>'Name') as org_name,
coalesce(m.policyholder->>'Addr',o.policyholder->>'Addr') as org_addr,
coalesce(m.policyholder->>'CreditCode',o.policyholder->>'CreditCode') as org_credit_code,
coalesce(m.policyholder->>'Contact',o.policyholder->>'Contact') as org_contact,
coalesce(m.policyholder->>'Phone',o.policyholder->>'Phone') as org_phone,
coalesce(m.policyholder->>'ContactRole',o.policyholder->>'ContactRole') as org_contact_role,
coalesce(m.policyholder->>'CreditCodePic',o.policyholder->>'CreditCodePic') as org_credit_code_pic,
coalesce(m.policyholder->>'SchoolCategory',o.policyholder->>'SchoolCategory') as org_school_category,
coalesce(m.policyholder->>'Province',o.policyholder->>'Province') as org_province,
coalesce(m.policyholder->>'City',o.policyholder->>'City') as org_city,
coalesce(m.policyholder->>'District',o.policyholder->>'District') as org_district,
coalesce(m.insured->>'Name',o.insured->>'Name') as insured_name,
coalesce(m.insured->>'Province',o.insured->>'Province') as insured_province,
coalesce(m.insured->>'City',o.insured->>'City') as insured_city,
coalesce(m.insured->>'District',o.insured->>'District') as insured_district,
coalesce((m.insured ->> 'IsCompulsory')::boolean,(o.insured ->> 'IsCompulsory')::boolean) as insured_is_compulsory,
coalesce(m.insured->>'Category',o.insured->>'Category') as insured_category,
coalesce(m.insured->>'SchoolCategory',o.insured->>'SchoolCategory') as insured_school_category,

-- 从m.insured_list中提取insert/update/delete的数据
o.insured_list as original_insured_list,
m.insured_list as insured_list,
(select jsonb_agg(list) from jsonb_array_elements(m.insured_list) as list
 where m.insured_list <> '{}' and list->>'Action'='insert') insert_insured_list,
(select jsonb_agg(list) from jsonb_array_elements(m.insured_list) as list
 where m.insured_list <> '{}' and list->>'Action'='delete') delete_insured_list,
(select jsonb_agg(list) from jsonb_array_elements(m.insured_list) as list
 where m.insured_list <> '{}' and list->>'Action'='update') update_insured_list,
--  从原清单中找出会被修改的数据
(select jsonb_agg(a_list) from jsonb_array_elements(o.insured_list) a_list where a_list->>'ID' in
 (select list->>'ID' from jsonb_array_elements(m.insured_list) as list
  where m.insured_list <> '{}' and list->>'Action'='update')) require_update_insured_list,

coalesce(m.policy_scheme,o.policy_scheme) as policy_scheme,
coalesce(m.activity_name,o.activity_name) as activity_name,
coalesce(m.activity_desc,o.activity_desc) as activity_desc,
coalesce(m.activity_location,o.activity_location) as activity_location,
coalesce(m.activity_date_set,o.activity_date_set) as activity_date_set,
coalesce(m.indate,o.indate) as indate,
o.policyholder||jsonb_strip_nulls(m.policyholder) as policyholder,
coalesce(m.policyholder_id,o.policyholder_id) as policyholder_id,
o.insured||jsonb_strip_nulls(m.insured) as insured,
coalesce(m.insured_id,o.insured_id) as insured_id,
coalesce(m.insured_group_by_day,o.insured_group_by_day) as insured_group_by_day,
coalesce(m.charge_mode,o.charge_mode) as charge_mode,
coalesce(m.amount,o.amount) as amount,
coalesce(m.insured_count,o.insured_count) as insured_count,
coalesce(m.insured_type,o.insured_type) as insured_type,
coalesce(m.non_compulsory_student_num,o.non_compulsory_student_num) as non_compulsory_student_num,
coalesce(m.compulsory_student_num,o.compulsory_student_num) as compulsory_student_num,
coalesce(m.canteen_num,o.canteen_num) as canteen_num,
coalesce(m.shop_num,o.shop_num) as shop_num,
coalesce(m.dinner_num,o.dinner_num) as dinner_num,
coalesce(m.pay_type,o.pay_type) as pay_type,
o.fee_scheme||jsonb_strip_nulls(m.fee_scheme) as fee_scheme,
coalesce(m.dispute_handling,o.dispute_handling) as dispute_handling,
coalesce(m.have_sudden_death,o.have_sudden_death) as have_sudden_death,
coalesce(m.prev_policy_no,o.prev_policy_no) as prev_policy_no,
coalesce(m.pool_name,o.pool_name) as pool_name,
coalesce(m.have_explosive,o.have_explosive) as have_explosive,
coalesce(m.have_rides,o.have_rides) as have_rides,
coalesce(m.inner_area,o.inner_area) as inner_area,
coalesce(m.outer_area,o.outer_area) as outer_area,
coalesce(m.traffic_num,o.traffic_num) as traffic_num,
coalesce(m.temperature_type,o.temperature_type) as temperature_type,
coalesce(m.open_pool_num,o.open_pool_num) as open_pool_num,
coalesce(m.heated_pool_num,o.heated_pool_num) as heated_pool_num,
coalesce(m.training_pool_num,o.training_pool_num) as training_pool_num,
coalesce(m.pool_num,o.pool_num) as pool_num,
coalesce(m.custom_type,o.custom_type) as custom_type,
coalesce(m.same,o.same) as same,
coalesce(m.arbitral_agency,o.arbitral_agency) as arbitral_agency,
case when m.files='{}' or m.files is null then o.files else m.files||o.files end as files,
coalesce(m.have_negotiated_price,o.have_negotiated_price) as have_negotiated_price,
 m.endorsement_status,
 m.application_files,
 m.need_balance,
 o.balance,
 o.balance_list,
 o.order_status,
(select string_agg(p.sn,',') from t_insurance_policy p where p.order_id = m.order_id) as sn,
 m.revoked_policy_no,
 m.policy_regen,
 m.clear_list,
 m.files_to_remove,
 m.invoice_header,
 m.correct_level,
 m.correct_log,
 m.refused_reason,
 m.addi,
 m.create_time,
 m.update_time,
 m.creator,
 m.remark,
 m.status
 from t_mistake_correct m
 join t_order o on o.id=m.order_id
 left JOIN t_insurance_types it ON it.id = o.insurance_type_id;

comment on view v_mistake_correct_show is
'v_mistake_correct_show';

drop table if exists t_v_mistake_correct_show;
create table t_v_mistake_correct_show as select * from v_mistake_correct_show limit 1;

/*==============================================================*/
/* View: v_order2                                               */
/*==============================================================*/
create or replace view v_order2 as
select o.id,
       o.trade_no,
       o.pay_order_no,
       o.insure_order_no,
       o.batch,
       o.create_time,
       o.pay_time,
       o.pay_channel,
       o.pay_type,
       o.pay_name,
       o.unit_price,
       o.amount,
       o.balance,
       o.balance_list,
       o.org_id,
       o.have_sudden_death,
       o.ground_num ,
       (case 
            when (policy_scheme ->> 'RefID' is null or policy_scheme->>'RefID' = '0') and (o.policy_scheme ->> 'ParentID')::int8 = 10040 THEN '散单用户'::text
            when policy_scheme ->> 'DataType'='8' and ((policy_scheme -> 'ParentID')::int8 in (10022,10024,10026,10028,10030)) then  '散单用户'
            else '团体投保'
       end) as plan_type,
       o.reminders_num,
       o.org_manager_id,
       o.insurance_type,
       o.have_dinner_num,
       o.have_confirm_date,
       o.insurance_type_id,o.health_survey,
       o.plan_id,o.plan_name,o.insurer,
       o.policy_scheme,
       o.policy_doc,
       o.activity_name,
       o.activity_category,
       o.activity_desc,
       o.activity_location,
       o.activity_date_set,
       o.copies_num,
       o.insured_count,
       o.compulsory_student_num,
       o.non_compulsory_student_num,
       o.contact,
       o.fee_scheme,
       o.car_service_target,
       o.policyholder,
       (o.insured_list -> 0)::jsonb ->> 'IDCardNo'                                      first_insured_id_card_no,
       o.policyholder ->> 'Addr'                                                     AS org_addr,
       o.policyholder ->> 'CreditCode'                                               AS org_credit_code,
       o.policyholder ->> 'Contact'                                                  AS org_contact,
       o.policyholder ->> 'Phone'                                                    AS org_phone,
       o.policyholder ->> 'ContactRole'                                              AS org_contact_role,
       o.policyholder ->> 'CreditCodePic'                                            AS org_credit_code_pic,
       o.policyholder ->> 'SchoolCategory'                                           AS org_school_category,
       o.policyholder ->> 'CompulsoryStudentNum'                                     AS org_compulsory_student_num,
       o.policyholder ->> 'NonCompulsoryStudentNum'                                  AS org_non_compulsory_student_num,
       o.policyholder_type,
       o.policyholder_id,
       o.same,
       o.relation,
       o.insured,
       o.insured ->> 'Name'                                                     AS insured_name,
       o.insured ->> 'Province'                                                 AS insured_province,
       o.insured ->> 'City'                                                     AS insured_city,
       o.insured ->> 'District'                                                 AS insured_district,
       (o.insured ->> 'IsCompulsory')::boolean                                  AS insured_is_compulsory,
       o.insured ->> 'Category'                                                 AS insured_category,
       o.insured ->> 'SchoolCategory'                                           AS insured_school_category,
       
       o.insured ->> 'PostCode' AS insured_post_code,
       o.insured ->> 'Phone' AS insured_phone,
       o.policy_scheme ->> 'Title' AS policy_scheme_title,

       o.policyholder->'BusinessDomain' as org_business_domain,
       o.insured->'BusinessDomain' as insured_business_domain,

       
       o.insured_id,
       o.have_insured_list,
       o.insured_group_by_day,
       o.insured_type,
       o.insured_list,
       o.commence_date,
       o.expiry_date,
       o.indate,
       o.sign,
       o.jurisdiction,
       o.dispute_handling,
       o.prev_policy_no,
       o.reminder_times,
       o.insure_base,
       o.blanket_insure_code,
       o.custom_type,
       o.train_projects,
       o.business_locations,
       o.training_pool_num, 
       o.heated_pool_num,
       o.open_pool_num,
       o.pool_num,
       o.dinner_num,
       o.canteen_num,
       o.shop_num,
       o.have_rides,
       o.have_explosive,
       o.inner_area+o.outer_area area,
       o.traffic_num,
       o.temperature_type,
       o.is_indoor,
       o.extra,
       o.bank_account,
       o.pay_contact,
       o.sudden_death_terms,
       o.spec_agreement,
       o.inner_area,
       o.outer_area,
       o.pool_name,
       o.arbitral_agency,
        o.confirm_refund,

       o.insured ->> 'CreditCode'        AS insured_credit_code,
       o.insured ->> 'Addr'              AS insured_addr,

       o.creator,
       o.domain_id,
       o.files                                                                       AS order_files,
       o.addi,
       o.remark,
       o.status,
       o.order_status,
       o.have_negotiated_price,
       o.lock_status,
       o.insurance_company,
       o.insurance_company_account,
       o.actual_amount,
       o.can_revoke_order,
       o.can_public_transfers,
       o.is_reminder,
       o.traits,



       o.is_invoice,
       o.inv_borrow,
       o.inv_visible,
       o.inv_title,
       o.inv_status,
       o.upd_status,


       (select sum(d.driver_seat_number) from t_insured_detail d where d.order_id = o.id) driver_seat_num,
       ----t_insured_detail中使用的是seat_num,approved_passengers_num字段弃用
       (select sum(d.seat_num) from t_insured_detail d where d.order_id = o.id) approved_passengers_num,


       o.refused_reason,
       o.unpaid_reason,
       o.updated_by,
       o.update_time,
       o.have_renewal_reminder,
       o.charge_mode,
       o.admin_received,
       o.user_received,
       (EXISTS ( SELECT p.id
           FROM t_insurance_policy p
          WHERE p.order_id = o.id)) AS have_policy,
          
       it.parent_id                                                                  AS insurance_type_parent_id,



       -- 能否修改发票抬头
       lock_status!='02' and (EXISTS ( SELECT p.id
           FROM t_insurance_policy p
          WHERE p.order_id = o.id)) and coalesce((select count(*) from t_mistake_correct c where c.order_id = o.id and modify_type = '4')
            <it.invoice_title_update_times,false) can_inv_title_modify,



       (select case
                   when it.parent_id = 10000 then (select it2.name from t_insurance_types it2 where it2.id = 10000)
                   when it.parent_id = 10040 then (select it2.name from t_insurance_types it2 where it2.id = 10040)
                   else (select o.insurance_type)
                   end)                                                              AS insurance_display,



        --更正次数
        (select count(*) from t_mistake_correct c where c.order_id = o.id and (modify_type = '2' or modify_type is null)) user_correct_times,
       -- 更正等级

       (select case
                   when it.parent_id = 10000 then (
                       select case
                                  when (expiry_date < (floor(date_part('epoch'::text, now() + interval '8 hour')) * 1000::double precision)::bigint)
                                      then (select '00')
                                  when o.pay_type is null or o.pay_type = '' then (select '00')
                                  when o.order_status = '24' or o.order_status = '28' then (select '00')
                                  when o.pay_type <> '在线支付' 
                                  then (
                                          select case 
                                          when o.order_status = '20' then (
                                                 select case when (commence_date > (floor(date_part('epoch'::text, now() + interval '8 hour')) * 1000::double precision)::bigint) then (select '22')
                                                        when (commence_date < (floor(date_part('epoch'::text, now() + interval '8 hour')) * 1000::double precision)::bigint) then (select '24')
                                                 end
                                          )
                                          when (commence_date > (floor(date_part('epoch'::text, now() + interval '8 hour')) * 1000::double precision)::bigint) then (select '20')
                                          when (commence_date <= (floor(date_part('epoch'::text, now() + interval '8 hour')) * 1000::double precision)::bigint) then (select '24')
                                          end
                                          )
                                  when o.pay_type = '在线支付' 
                                  then (
                                         select case 
                                          when (o.order_status = '16' or o.order_status = '18') and (commence_date > (floor(date_part('epoch'::text, now() + interval '8 hour')) * 1000::double precision)::bigint) then (select '20')
                                          when o.order_status <> '20' then (select '00')
                                          when (commence_date > (floor(date_part('epoch'::text, now() + interval '8 hour')) * 1000::double precision)::bigint) then (select '22')
                                          when (commence_date <= (floor(date_part('epoch'::text, now() + interval '8 hour')) * 1000::double precision)::bigint) then (select '24')
                                          end
                                          ) 
                                  else (select '00')
                                  end)
--  学意险
                   when it.parent_id = 10040 then (
                       select case
                                  when (expiry_date <
                                        (floor(date_part('epoch'::text, now() + interval '8 hour')) * 1000::double precision)::bigint)
                                      then (select '00')
                                  else (select '20') end)
-- 校方系列
                   when it.parent_id = 10020 or it.id = 10060 or it.id = 10070 or it.id = 10080 then (
                       select case
                                  when (expiry_date <
                                        (floor(date_part('epoch'::text, now() + interval '8 hour')) * 1000::double precision)::bigint or
                                        lock_status = '04') then (select '00')
                                  when (select string_agg(p.sn, ',')
                                        from t_insurance_policy p
                                        where p.order_id = o.id) isnull then (select '20')
                                  when (select string_agg(p.sn, ',')
                                        from t_insurance_policy p
                                        where p.order_id = o.id) notnull then (select '22')
                                  else (select '00')
                                  end
                   )
                   end ::varchar)                                                    AS correct_level,

--        投保单位信息（JSON提取方便查询）
       o.policyholder ->> 'Name'                                                     AS org_name,
       o.policyholder ->> 'Province'                                                 AS org_province,
       o.policyholder ->> 'City'                                                     AS org_city,
       o.policyholder ->> 'District'                                                 AS org_district,
       o.policyholder ->> 'IsCompulsory'                                             AS org_is_compulsory,
       o.policyholder ->> 'IsSchool'                                                 AS org_is_school,
       coalesce((select array_length(string_to_array(o.reminder_times, ','), 1)), 0) as reminder_times_count,
-- 被保险人（个人）
       u.official_name                                                               AS i_official_name,
       u.id_card_type                                                                AS i_id_card_type,
       u.id_card_no                                                                  AS i_id_card_no,
       u.mobile_phone                                                                AS i_mobile_phone,
       u.gender                                                                      AS i_gender,
       u.birthday                                                                    AS i_birthday,
       u.addi                                                                        AS i_addi,
-- 投保人（个人）
       h.official_name                                                               AS h_official_name,
       h.id_card_type                                                                AS h_id_card_type,
       h.id_card_no                                                                  AS h_id_card_no,
       h.mobile_phone                                                                AS h_mobile_phone,
       h.addi                                                                        AS h_addi,
-- 校快保用户
       x.subdistrict,
       x.faculty,
       x.grade,
       x.class,
       x.create_time                                                                 AS x_create_time,
-- 一期学校机构
       s.name                                                                        AS school,
       s.faculty                                                                     AS s_faculty,
       s.branches                                                                    AS s_branches,
       s.category                                                                    AS s_category,
       s.province,
       s.city,
       s.district,
       s.data_sync_target,
       s.sale_managers,
       s.school_managers,
       s.purchase_rule,
       s.create_time                                                                 AS s_create_time,
--   保单
       (o.actual_amount - o.amount)                                                  as difference,
       (select string_agg(case
                              when (p.status = '00' or p.status = '12') and p.sn is not null then p.sn
                              when p.status = '04' or p.status = '08' then '--'
                              when p.sn is null then '--'
                              end, '\n')
        from t_insurance_policy p
        where order_id = o.id
          and p.status not in ('16', '20', '24'))                                    AS policy_no,

--  缴费状态 只有订单下所有保单都是已缴费(is_admin_pay = true)，才是已缴费。(线上线下处理方式一样)
       (with admin_pay_count as (select count(*) count
                                 from t_insurance_policy p
                                 where order_id = o.id
                                   and is_admin_pay = true
                                   and p.status not in ('16', '20', '24')),
             policy_count as (select count(*) count
                              from t_insurance_policy p
                              where order_id = o.id
                                and p.status not in ('16', '20', '24'))
        select case
                   when (select count from policy_count) > 0 and
                        (select count from admin_pay_count) = (select count from policy_count)
                       then '已缴费'
                   else '未缴费'
                   end
       )                                                                             as fee_status

from t_order o
         left JOIN t_insurance_types it ON it.id = o.insurance_type_id
         left JOIN t_school s ON o.org_id = s.id --一期学校信息 s
         left JOIN t_user u ON o.insured_id = u.id --被保险人（个人）信息 u
         left JOIN t_user h ON o.policyholder_id = h.id --投保人（个人）信息 h
         left JOIN t_xkb_user x on o.insured_id = x.id;

comment on view v_order2 is
'二期订单视图，兼容一期';

drop table if exists t_v_order2;
create table t_v_order2 as select * from v_order2 limit 1;

/*==============================================================*/
/* View: v_order_sum                                            */
/*==============================================================*/
create or replace view v_order_sum as
select 
  x.school,x.org_id,x.batch,x.order_number,x.order_amount,
  y.cancel_number,y.cancel_amount
from 
(select school,org_id,batch,count(id) as order_number, sum(amount)/100 as order_amount
from v_order 
  where status='4' and amount >=100 
  group by school,org_id,batch
  order by org_id asc,batch desc) x left join
(select school,org_id,batch,count(id) as cancel_number, sum(amount)/100 as cancel_amount
from v_order 
where status='6' and amount >=100 and pay_time is not null
  group by school,org_id,batch
  order by org_id asc,batch desc) y
  on x.batch=y.batch;

comment on view v_order_sum is
'v_order_sum';

drop table if exists t_v_order_sum;
create table t_v_order_sum as select * from v_order_sum limit 1;

/*==============================================================*/
/* View: v_paper                                                */
/*==============================================================*/
create or replace view v_paper as
WITH paper_basic AS (
         SELECT p.id AS paper_id,
            p.domain_id,
            p.exampaper_id,
            p.name,
            p.assembly_type,
            p.category,
            p.level,
            p.suggested_duration,
            p.description,
            p.tags,
            p.config,
            p.creator,
            jsonb_build_object('id', u.id, 'official_name', u.official_name, 'account', u.account, 'mobile_phone', u.mobile_phone, 'email', u.email) AS creator_info,
            p.create_time,
            p.updated_by,
            p.update_time,
            p.version,
            p.status AS paper_status
           FROM t_paper p
             LEFT JOIN t_user u ON p.creator = u.id
        ),
        -- 原试卷题组
        paper_valid_groups AS (
         SELECT pg.id,
            pg.paper_id,
            pg.name,
            pg."order",
            pg.creator,
            pg.create_time,
            pg.updated_by,
            pg.update_time,
            pg.status,
            pg.addi
           FROM t_paper_group pg
          WHERE pg.status::text <> '02'::text
        ),
        -- 原题库题目（聚合时包含分数，供统计复用）
        group_valid_questions AS (
         SELECT pq.group_id,
            sum(pq.score) AS group_total_score,
            count(pq.id) AS group_question_count,
            jsonb_agg(
                jsonb_build_object(
                    'id', pq.id, 
                    'bank_question_id', q.id, 
                    'type', q.type, 
                    'content', q.content, 
                    'options', q.options, 
                    'answers', q.answers, 
                    'score', pq.score, 
                    'sub_score', pq.sub_score, 
                    'difficulty', q.difficulty, 
                    'tags', q.tags, 
                    'analysis', q.analysis, 
                    'title', q.title, 
                    'answer_file_path', q.answer_file_path, 
                    'test_file_path', q.test_file_path, 
                    'input', q.input, 
                    'output', q.output, 
                    'example', q.example, 
                    'repo', q.repo, 
                    'order', pq."order", 
                    'group_id', pq.group_id, 
                    'status', q.status, 
                    'question_attachments_path', q.question_attachments_path
                ) ORDER BY pq."order"
            ) AS questions
           FROM t_paper_question pq
             JOIN t_question q ON pq.bank_question_id = q.id AND q.status::text = '00'::text
          WHERE pq.status::text <> '02'::text
          GROUP BY pq.group_id
        ),
        -- 考卷题组
        exam_valid_groups AS (
         SELECT eg.id,
            eg.exam_paper_id,
            eg.name,
            eg."order",
            eg.creator,
            eg.create_time,
            eg.updated_by,
            eg.update_time,
            eg.status,
            eg.addi
           FROM t_exam_paper_group eg
          WHERE eg.status::text <> '02'::text
        ),
        -- 考卷题目（优化：仅按group_id分组）
        exam_group_valid_questions AS (
         SELECT eq.group_id,
            sum(eq.score) AS group_total_score,
            count(eq.id) AS group_question_count,
            jsonb_agg(
                jsonb_build_object(
                    'id', eq.id, 
                    'type', eq.type, 
                    'content', eq.content, 
                    'options', eq.options, 
                    'answers', eq.answers, 
                    'score', eq.score, 
                    'order', eq."order", 
                    'group_id', eq.group_id, 
                    'status', eq.status, 
                    'analysis', eq.analysis, 
                    'title', eq.title, 
                    'answer_file_path', eq.answer_file_path, 
                    'test_file_path', eq.test_file_path, 
                    'input', eq.input, 
                    'output', eq.output, 
                    'example', eq.example, 
                    'repo', eq.repo, 
                    'commit_id', eq.commit_id,
                    'question_attachments_path', eq.question_attachments_path
                ) ORDER BY eq."order"
            ) AS questions
           FROM t_exam_paper_question eq
          WHERE eq.status::text <> '02'::text
          GROUP BY eq.group_id
        ),
        -- 考卷统计数据（复用题组题目聚合结果）
        exam_stats_agg AS (
         SELECT 
            eg.exam_paper_id,
            COUNT(DISTINCT eg.id) AS group_count,
            SUM(eq.group_question_count) AS question_count,
            SUM(eq.group_total_score) AS total_score
         FROM exam_valid_groups eg
         LEFT JOIN exam_group_valid_questions eq ON eg.id = eq.group_id
         GROUP BY eg.exam_paper_id
        ),
        -- 条件选择题组和题目数据源（优化：用LATERAL替代子查询）
        group_with_questions AS (
         SELECT pb.paper_id,
            COALESCE(exam_groups_data, original_groups_data) AS groups_data
         FROM paper_basic pb
         -- 考卷场景数据
         LEFT JOIN LATERAL (
            SELECT jsonb_agg(
                jsonb_build_object(
                    'id', eg.id, 
                    'name', eg.name, 
                    'order', eg."order", 
                    'creator', eg.creator, 
                    'create_time', eg.create_time, 
                    'updated_by', eg.updated_by, 
                    'update_time', eg.update_time, 
                    'status', eg.status, 
                    'addi', eg.addi, 
                    'questions', COALESCE(eq.questions, '[]'::jsonb)
                ) ORDER BY eg."order"
            ) AS exam_groups_data
            FROM exam_valid_groups eg
            LEFT JOIN exam_group_valid_questions eq ON eg.id = eq.group_id
            WHERE eg.exam_paper_id = pb.exampaper_id
              AND pb.paper_status <> '00'::text AND pb.exampaper_id IS NOT NULL
         ) AS exam_data ON true
         -- 原题库场景数据
         LEFT JOIN LATERAL (
            SELECT jsonb_agg(
                jsonb_build_object(
                    'id', pg.id, 
                    'name', pg.name, 
                    'order', pg."order", 
                    'creator', pg.creator, 
                    'create_time', pg.create_time, 
                    'updated_by', pg.updated_by, 
                    'update_time', pg.update_time, 
                    'status', pg.status, 
                    'addi', pg.addi, 
                    'questions', COALESCE(q.questions, '[]'::jsonb)
                ) ORDER BY pg."order"
            ) AS original_groups_data
            FROM paper_valid_groups pg
            LEFT JOIN group_valid_questions q ON pg.id = q.group_id
            WHERE pg.paper_id = pb.paper_id
              AND (pb.paper_status = '00'::text OR pb.exampaper_id IS NULL)
         ) AS original_data ON true
        ),
        -- 试卷统计数据（优化：复用题组聚合结果，减少JOIN）
        paper_stats AS (
         SELECT pb.paper_id,
            CASE 
                WHEN pb.paper_status <> '00'::text AND pb.exampaper_id IS NOT NULL THEN
                    COALESCE(MAX(es.total_score), 0::double precision)
                ELSE
                    COALESCE(SUM(q.group_total_score), 0::double precision)
            END AS total_score,
            CASE 
                WHEN pb.paper_status <> '00'::text AND pb.exampaper_id IS NOT NULL THEN
                    COALESCE(MAX(es.question_count), 0)
                ELSE
                    COALESCE(SUM(q.group_question_count), 0)
            END AS question_count,
            CASE 
                WHEN pb.paper_status <> '00'::text AND pb.exampaper_id IS NOT NULL THEN
                    COALESCE(MAX(es.group_count), 0)
                ELSE
                    COUNT(DISTINCT pg.id)
            END AS group_count
           FROM paper_basic pb
           -- 原题库场景：关联题组和已聚合的题目数据
           LEFT JOIN paper_valid_groups pg ON pb.paper_id = pg.paper_id
           LEFT JOIN group_valid_questions q ON pg.id = q.group_id
           -- 考卷场景：关联预计算的统计结果
           LEFT JOIN exam_stats_agg es ON pb.exampaper_id = es.exam_paper_id
          GROUP BY pb.paper_id, pb.exampaper_id, pb.paper_status
        )
 -- 最终查询
 SELECT p.paper_id AS id,
    p.domain_id,
    p.exampaper_id,
    p.name,
    p.assembly_type,
    p.category,
    p.level,
    p.suggested_duration,
    p.description,
    p.tags,
    p.config,
    p.creator,
    p.creator_info,
    p.create_time,
    p.updated_by,
    p.update_time,
    p.version,
    p.paper_status AS status,
    s.total_score,
    s.question_count,
    s.group_count,
    COALESCE(g.groups_data, '[]'::jsonb) AS groups_data
   FROM paper_basic p
     JOIN paper_stats s ON p.paper_id = s.paper_id
     LEFT JOIN group_with_questions g ON p.paper_id = g.paper_id
  ORDER BY p.domain_id, p.update_time;

comment on view v_paper is
'试卷';

drop table if exists t_v_paper;

create table t_v_paper as select * from v_paper;

/*==============================================================*/
/* View: v_param                                                */
/*==============================================================*/
create or replace view v_param as
with recursive parent as (
select 
	p.belongto,
	p.id,
	p.name,
	p.data_type,
	p.value,
    p.remark,
    p.status
from t_param p
where belongto=0
union
select 
  child.belongto,
	child.id,
	child.name,
	child.data_type,
	child.value,
    child.remark,
    child.status
from t_param child
join parent on child.belongto=parent.id)
select 
    a.belongto parent_ID,
    b.name parent_Name,
    a.id,
    a.name,
    a.data_type,
    a.value,
    a.remark,
    a.status 
from parent a join t_param b on a.belongto=b.id;

comment on view v_param is
'v_param';

drop table if exists t_v_param;
create table t_v_param as select * from v_param;

/*==============================================================*/
/* View: v_payment                                              */
/*==============================================================*/
create or replace view v_payment as
select pay.*,
       p.is_admin_pay,
       p.premium,
       p.third_party_premium,
       p.policyholder ->> 'Name'::varchar as policyholder_name
from t_payment pay
left join t_insurance_policy p on pay.policy_no = p.sn and p.status = '00';

comment on view v_payment is
'v_payment';

drop table if exists t_v_payment;
create table t_v_payment as select * from v_payment limit 1;

/*==============================================================*/
/* View: v_practice_unmarked_student_cnt                        */
/*==============================================================*/
create or replace view v_practice_unmarked_student_cnt as
SELECT p.id                                       AS practice_id,
       COALESCE(count(DISTINCT ps.id), 0::bigint) AS unmarked_count
FROM t_practice p
         LEFT JOIN t_practice_submissions ps ON ps.practice_id = p.id AND ps.status::text = '06'::text
GROUP BY p.id;

comment on view v_practice_unmarked_student_cnt is
'练习待批改人数统计视图';

drop table if exists t_v_practice_unmarked_student_cnt;

create table t_v_practice_unmarked_student_cnt as select * from v_practice_unmarked_student_cnt;

/*==============================================================*/
/* View: v_question_bank                                        */
/*==============================================================*/
create or replace view v_question_bank as
SELECT
    b.id,
    b.domain_id,
    b.name,
    b.type,
    b.tags,
    b.creator,
    u.official_name,
    b.create_time,
    b.update_time,
    COUNT(DISTINCT q.id) AS question_count,
    COALESCE(array_agg(DISTINCT q.type) FILTER (WHERE q.type IS NOT NULL), ARRAY[]::text[]) as question_types,
    COALESCE(array_agg(DISTINCT q.difficulty) FILTER (WHERE q.difficulty IS NOT NULL), ARRAY[]::bigint[]) as question_difficulties,
    COALESCE(
        array_agg(DISTINCT tag) FILTER (WHERE tag IS NOT NULL),
        ARRAY[]::text[]
    ) as question_tags
FROM
    t_question_bank b
LEFT JOIN
    t_user u ON b.creator = u.id
LEFT JOIN
    t_question q ON b.id = q.belong_to AND q.status = '00'
LEFT JOIN LATERAL
    jsonb_array_elements_text(q.tags) as tag ON true
WHERE
    b.status = '00'
GROUP BY
    -- 所有非聚合字段都必须出现在这里
    b.id,
    b.domain_id,
    b.name,
    b.type,
    b.tags,
    b.creator,
    u.official_name,  -- 添加遗漏的u.official_name
    b.create_time,    -- 添加遗漏的b.create_time
    b.update_time;

comment on view v_question_bank is
'v_question_bank';

drop table if exists t_v_question_bank;

create table t_v_question_bank as select * from v_question_bank;

/*==============================================================*/
/* View: v_region                                               */
/*==============================================================*/
create or replace view v_region as
select
p.region_name province,
c.region_name city,
d.region_name district,
s.region_name street
from t_region p
left join t_region c on c.parent_id=p.id and c.level=4 and c.region_name <>'中山市' and c.region_name<>'东莞市'
left join t_region d on d.parent_id=c.id and d.level=6
left join t_region s on s.parent_id=d.id and s.level=8
where p.parent_id=0
union
select
   '广东省',
c.region_name city,
null,
s.region_name street
from t_region c
left join t_region s on s.parent_id=c.id and s.level=8
where c.region_name='中山市' or c.region_name = '东莞市';

comment on view v_region is
'v_region';

drop table if exists t_v_region;
create table t_v_region as select * from v_region  limit 1;

/*==============================================================*/
/* View: v_report_claims                                        */
/*==============================================================*/
create or replace view v_report_claims as
select 
r.id,
r.informant_id,
r.informant,
r.insured_id,
r.insured,
r.insurance_type,
r.insurance_policy_sn,
r.insurance_policy_id,
i.order_id,
o.org_id,
o.plan_id,
a.policy_no,
r.insurance_policy_start,
r.insurance_policy_cease,
r.report_sn,
r.insured_channel,
r.insured_org,
r.treatment,
r.hospital,
r.injured_location,
r.injured_part,
r.reason,
r.injured_desc,
r.credit_code,
r.bank_account_type,
r.bank_account_name,
r.bank_name,
r.bank_account_id,
r.bank_card_pic,
r.injured_id_pic,
r.guardian_id_pic,
r.org_lic_pic,
r.relation_prove_pic,
r.bills_pic,
r.hospitalized_bills_pic,
r.invoice_pic,
r.medical_record_pic,
r.dignostic_inspection_pic,
r.discharge_abstract_pic,
r.other_pic,
r.courier_sn_pic,
r.paid_notice_pic,
r.claim_apply_pic,
r.equity_transfer_file,
r.match_programme_pic,
r.policy_file,
r.addi_pic,
r.courier_sn,
r.reply_addr,
r.injured_time,
r.report_time,
r.reply_time,
r.claims_mat_add_time,
r.mat_return_date,
r.close_date,
r.face_amount,
r.medi_assure_amount,
r.third_pay_amount,
r.claim_amount,
r.refuse_desc,
r.addi,
r.creator,
r.domain_id,
r.remark,
r.status,
s.name school,
s.category school_type,
b.grade,
b.class,
u.official_name,
u.gender,
u.id_card_type,
u.id_card_no,
u.birthday,
r.insured->>'official_name' w_offcial_name,
r.insured->>'id_card_type' w_id_card_type,
r.insured->>'id_card_no' w_id_card_no,
r.insured->>'OfficialName' insured_offcial_name,
m.official_name m_offcial_name,
m.gender m_gender,
m.id_card_type m_id_card_type,
m.id_card_no m_id_card_no,
m.mobile_phone m_mobile_phone,
r.informant->>'official_name' w_m_offcial_name,
r.informant->>'mobile_phone' w_m_mobile_phone,

COALESCE(r.insurance_type_id, o.insurance_type_id) AS insurance_type_id,

it.parent_id as insurance_type_parent_id,
i.sn,
r.occurr_reason,
r.treatment_result,
r.disease_diagnosis_pic,
r.disability_certificate,
r.death_certificate,
r.student_status_certificate
from t_report_claims r
left join t_user m on r.informant_id=m.id
left join t_user u on r.insured_id=u.id
left join t_wx_user c on u.id=c.id
left join t_xkb_user b on  u.id = b.id
left join t_school s on  b.school_id = s.id
left join t_insurance_policy i on i.id=r.insurance_policy_id
left join t_order o on i.order_id=o.id
left join t_insure_attach a on 
    CASE
            WHEN r.insurance_type_id = ANY (ARRAY[12002::bigint, 12004::bigint, 12006::bigint]) THEN o.org_id = a.school_id and o.batch=a.batch and b.grade=a.grade and a.year=i.year
            ELSE i.id = a.insure_policy_id
    END
left join t_insurance_types it on r.insurance_type_id = it.id;

comment on view v_report_claims is
'报案理赔视图';

drop table if exists t_v_report_claims;
create table t_v_report_claims as select * from v_report_claims limit 1;

/*==============================================================*/
/* View: v_student_answer_question                              */
/*==============================================================*/
create or replace view v_student_answer_question as
SELECT p.exam_session_id,
       p.practice_id,
       q.id  AS question_id,
       q."order",
       q.type,
       sa.answer,
       sa.answer_score,
       sa.actual_answers,
       sa.actual_options,
       e.id  AS examinee_id,
       ps.id AS practice_submission_id
FROM t_student_answers sa
         LEFT JOIN t_examinee e ON e.id = sa.examinee_id
         LEFT JOIN t_practice_submissions ps ON ps.id = sa.practice_submission_id
         JOIN t_exam_paper p ON p.exam_session_id = e.exam_session_id OR p.practice_id = ps.practice_id
         JOIN t_paper_group pg ON pg.paper_id = p.id
         JOIN t_exam_paper_question q ON q.group_id = p.id
WHERE e.status::text = '10'::text
   OR ps.status::text <> '04'::text;

comment on view v_student_answer_question is
'考试/练习-学生作答题目视图';

drop table if exists t_v_student_answer_question;

create table t_v_student_answer_question as select * from v_student_answer_question;

/*==============================================================*/
/* View: v_student_exam_total_score                             */
/*==============================================================*/
create or replace view v_student_exam_total_score as
 SELECT te.student_id,
    te.id AS examinee_id,
    te.exam_session_id,
    es.exam_id,
    e.name,
    e.submitted,
    es.start_time,
    es.end_time,
    sum(ta.answer_score) AS total_score,
    es.status
   FROM t_examinee te
     JOIN t_student_answers ta ON te.id = ta.examinee_id
     JOIN t_exam_session es ON te.exam_session_id = es.id
     JOIN t_exam_info e ON es.exam_id = e.id
  WHERE ta.type::text = '00'::text
  GROUP BY e.name, te.student_id, te.exam_session_id, es.exam_id, e.submitted, es.start_time, es.end_time, es.status, te.id;

comment on view v_student_exam_total_score is
'学生考试成绩视图';

drop table if exists t_v_student_exam_total_score;
create table t_v_student_exam_total_score as select * from v_student_exam_total_score;

/*==============================================================*/
/* View: v_student_practice_total_score                         */
/*==============================================================*/
create or replace view v_student_practice_total_score as
 SELECT DISTINCT ps.id,
    ps.practice_id,
    p.name,
    ps.student_id,
    ps.exam_paper_id,
        CASE
            WHEN bool_or(a.answer_score IS NULL) THEN NULL::double precision
            ELSE sum(a.answer_score)
        END AS total_score,
    p.type,
    ps.attempt,
    ps.status,
    count(
        CASE
            WHEN a.answer_score <> epq.score THEN 1
            ELSE NULL::integer
        END) AS wrong_count,
    (ps.end_time - ps.start_time) AS used_time
   FROM t_practice_submissions ps
     JOIN t_student_answers a ON ps.id = a.practice_submission_id
     JOIN t_exam_paper_question epq ON a.question_id = epq.id
     JOIN t_practice p ON ps.practice_id = p.id
  WHERE ps.status::text = '08'::text
  GROUP BY ps.id, p.id, ps.attempt, ps.student_id, ps.end_time, ps.start_time;

comment on view v_student_practice_total_score is
'展示学生练习成绩视图                      ';

drop table if exists t_v_student_practice_total_score;
create table t_v_student_practice_total_score as select * from v_student_practice_total_score;

/*==============================================================*/
/* View: v_user                                                 */
/*==============================================================*/
create or replace view v_user as
select
	u.ID,
	u.External_ID_Type,
	u.External_ID,
	u.Category,
	u.Type,
	u.Language,
	u.Country,
	u.Province,
	u.City,
	u.Addr,
	u.Official_Name,
	u.ID_Card_Type,
	u.ID_Card_No,
	u.Mobile_Phone,
	u.Email,
	u.Account,
	u.Gender,
	u.Birthday,
	u.Nickname,
	u.Avatar,
	u.Avatar_Type,
	u.Dev_ID,
	u.Dev_User_ID,
	u.Dev_Account,
	u.IP,
	u.Port,
	u.Auth_Failed_Count,
	u.Lock_Duration,
	u.Visit_Count,
	u.Attack_Count,
	u.Lock_Reason,
	u.Logon_Time,
	u.Begin_Lock_Time,
	u.Creator,
	u.Create_Time,
	u.updated_by,
	u.Update_Time,
	u.Domain_ID,
	u.Addi,
	u.Remark,
	u.Status,
	wx.Wx_Open_ID,
	wx.Mp_Open_ID,
	wx.Union_ID,
	wx.Open_ID,
	wx.Nickname wx_nickname,
	wx.Head_Img_URL,
	wx.Create_Time wx_create_time,
	wx.Update_Time wx_update_time,
	g.id grp_id,
    g.realm,
	g.Name grp_name
from 
	t_user u
	left join t_wx_user wx on u.id=wx.id
	left join t_user_group ug on u.id=ug.user_id
	left join t_group g on ug.group_id=g.id;

comment on view v_user is
'整合t_user
t_wx_user(微信用户信息，open_id,nickname,等)
t_group(组名称)
t_user_group';

drop table if exists t_v_user;
create table if not exists t_v_user as select * from v_user limit 1;

/*==============================================================*/
/* View: v_user_domain                                          */
/*==============================================================*/
create or replace view v_user_domain as
select 
    ud.id,
    u.id as user_id,
    coalesce(
        u.official_name, 
        u.nickname, 
        u.mobile_phone, 
        u.account, 
        u.id::text::character varying) AS user_name,
    u.mobile_phone,
    u.email,u.id_card_no,u.id_card_type,u.external_id,u.external_id_type,
    d.id as auth_domain_id,d.priority,
    d.name as domain_name,
    d.domain,
    ud.grant_source,
    ud.data_access_mode,
    ud.data_scope,ud.domain_id,
    ud.create_time,ud.remark,ud.addi,ud.creator,ud.status
from t_user_domain ud
  join t_user u on u.id=ud.sys_user
  join t_domain d on ud.domain=d.id
  order by u.id,d.id;

comment on view v_user_domain is
'user domain';

drop table if exists t_v_user_domain;
create table t_v_user_domain as select * from v_user_domain limit 1;

/*==============================================================*/
/* View: v_user_domain_api                                      */
/*==============================================================*/
create or replace view v_user_domain_api as
select 
	u.id as user_id,u.official_name,coalesce(u.official_name,u.nickname,u.mobile_phone,
	  u.account,u.id::text) as user_name,u.role,u.mobile_phone as mobile_phone,
		
     a.id as api_id,a.name as api_name,a.expose_path as api_expose_path,
	
	d.name as domain_name,d.id as domain_id, d.domain,	d.priority,
	
	ud.id as user_domain_id,
		ud.grant_source as user_domain_grant_source,
		ud.data_access_mode as user_domain_data_access_mode,		
		ud.data_scope as user_domain_data_scope,
		ud.data_scope->>'data' as user_domain_data_scope_data,
		ud.data_scope->>'type' as user_domain_data_scope_type,
		ud.id_on_domain,
		ud.create_time as user_domain_create_time,
		
	da.id as domain_api_id,
		da.grant_source as domain_api_grant_source,
		da.data_access_mode as domain_api_data_access_mode,		
		da.data_scope as domain_api_data_scope,
		da.data_scope->>'data' as domain_api_data_scope_data,
		da.data_scope->>'type' as domain_api_data_scope_type,
		da.create_time as domain_api_create_time
from t_user_domain ud
	join t_domain_api da on ud.domain=da.domain
	join t_user u on ud.sys_user = u.id 
	join t_domain d on ud.domain=d.id
	join t_api a on da.api=a.id
order by ud.sys_user,ud.domain,da.api;

comment on view v_user_domain_api is
'该视图代表用户、角色与功能三者的关系，即t_user,t_user_domain,t_domain,t_domain_api,t_api, 
该表以t_user_domain为主表，inner join t_domain_api 所以，如果用户没有角色，或者用户具有的角色没有权限，则该表中就没有该用户的数据，表示用户没有权限使用系统 。

该视图以ud.sys_user,ud.domain,da.api排序，即用户、角色与API次序排序。';

drop table if exists t_v_user_domain_api;
create table if not exists t_v_user_domain_api as select * from v_user_domain_api limit 1;

/*==============================================================*/
/* View: v_x_grade_list                                         */
/*==============================================================*/
create or replace view v_x_grade_list as
 SELECT ei.id AS exam_id,
    ei.name AS exam_name,
    ei.type AS exam_type,
    ets.exam_session_id,
    ets.total_score,
    es.start_time,
    es.end_time,
    es.mark_mode,
    ep.id AS exam_paper_id,
    ep.name AS paper_name
   FROM t_exam_info ei
     JOIN v_student_exam_total_score ets ON ei.id = ets.exam_id
     LEFT JOIN t_exam_session es ON es.id = ets.exam_session_id
     LEFT JOIN t_exam_paper ep ON ep.exam_session_id = ets.exam_session_id;

comment on view v_x_grade_list is
'v_x_grade_list';

drop table if exists t_v_grade_list;
create table t_v_grade_list as select * from v_x_grade_list;

/*==============================================================*/
/* View: v_y_max_submitted_view                                 */
/*==============================================================*/
create or replace view v_y_max_submitted_view as
 SELECT DISTINCT ON (practice_id, student_id) id,
    student_id,
    attempt,
    wrong_count,
    total_score,
    practice_id
   FROM v_student_practice_total_score
  ORDER BY practice_id, student_id, attempt DESC;

comment on view v_y_max_submitted_view is
'v_y_max_submitted_view';

 drop table if exists t_v_max_submitted_view;
create table t_v_max_submitted_view as select * from v_y_max_submitted_view;

/*==============================================================*/
/* View: v_z_grade_exam_session_info                            */
/*==============================================================*/
create or replace view v_z_grade_exam_session_info as
WITH pass_examinee AS (
    SELECT
        ets.exam_session_id,
        COUNT(DISTINCT ets.student_id) AS pass_examinees,
        vep.total_score
    FROM v_student_exam_total_score ets                                                           -- 展示学生考试成绩视图：考试场次ID 学生ID
        JOIN v_exam_paper vep ON vep.exam_session_id = ets.exam_session_id                        -- 该考试场次的成绩 v_exam_paper
    WHERE ets.total_score >= 0.6 * vep.total_score
    GROUP BY
        ets.exam_session_id,
        vep.id,
        vep.total_score
    ), examinee AS (
        SELECT
            exam_session_id,                                                                          -- 考试场次ID  
            COUNT(DISTINCT student_id) AS scheduled_examinees,                                        -- 计划参加考试的考生人数
            COUNT(
                CASE
                    WHEN status != '02' AND student_id IS NOT NULL THEN 1
                    ELSE NULL::integer
                END) AS actual_examinees                                                              -- 实际参加考试的考生人数
        FROM t_examinee                                                                               -- 从考生表：考试场次ID 计划参加考试的考生人数 实际参加考试的考生人数
        WHERE status <> '08'
        GROUP BY
            exam_session_id
        ORDER BY
            exam_session_id DESC
    )
SELECT
    es.exam_id,
    es.id AS exam_session_id,
    AVG(ets.total_score) AS average_score,
    es.start_time AS start_time,
    es.end_time AS end_time,
    es.mark_mode AS mark_mode,
    vep.id AS exam_paper_id,
    vep.name AS paper_name,
    vep.total_score AS total_score,
    ee.actual_examinees,
    ee.scheduled_examinees,
    ps.pass_examinees
FROM t_exam_session es                                                     -- 展示学生考试成绩视图：考试场次ID 学生ID 考试成绩 
    LEFT JOIN v_student_exam_total_score ets ON es.id = ets.exam_session_id                          -- 考试场次表：开始时间 结束时间 批改模式
    LEFT JOIN v_exam_paper vep ON vep.exam_session_id = es.id              -- 考试考卷表：试卷ID 试卷名称 试卷总分
    LEFT JOIN pass_examinee ps ON ps.exam_session_id = es.id
    LEFT JOIN examinee ee ON ee.exam_session_id = es.id                       -- 该考试场次的计划参加考试的考生人数 实际参加考试的考生人数
    GROUP BY
        ets.exam_id,
        ets.exam_session_id,
        es.id,
        vep.id,
        vep.name,
        vep.total_score,
        ee.actual_examinees,
        ee.scheduled_examinees,
        ps.pass_examinees
ORDER BY
    es.start_time DESC;

comment on view v_z_grade_exam_session_info is
'v_z_grade_exam_session_info';

/*==============================================================*/
/* View: v_z_grade_practice_statistics                          */
/*==============================================================*/
create or replace view v_z_grade_practice_statistics as
WITH pass_examinee AS (
    SELECT
        pts.practice_id,
        COUNT(DISTINCT pts.student_id) AS pass_num
    FROM v_student_practice_total_score pts                                 -- 展示学生成绩：练习ID 通过人数
        LEFT JOIN v_exam_paper vep ON vep.practice_id = pts.practice_id     -- 该练习的总分
    WHERE pts.total_score >= 0.6 * vep.total_score::integer
    GROUP BY
        pts.practice_id,
        vep.total_score
)
SELECT
    p.id AS practice_id,
    p.name AS practice_name,
    p.creator AS creator,
    vep.total_score AS total_score,
    ps.pass_num AS pass_student,
    AVG(pts.total_score) AS averge_score,
    COUNT(DISTINCT pts.student_id) AS actual_completer
FROM t_practice p                                                            -- 展示练习列表：练习ID 名字
    LEFT JOIN v_student_practice_total_score pts ON pts.practice_id = p.id   -- 该练习的平均成绩、完成人数
    LEFT JOIN v_exam_paper vep ON vep.practice_id = p.id                     -- 该练习的总分
    LEFT JOIN pass_examinee ps ON ps.practice_id = p.id                      -- 该练习的通过人数
GROUP BY
    p.id,
    p.name,
    p.creator,
    vep.total_score,
    ps.pass_num;

comment on view v_z_grade_practice_statistics is
'v_z_grade_practice_statistics';

/*==============================================================*/
/* View: v_z_practice_summary                                   */
/*==============================================================*/
create or replace view v_z_practice_summary as
SELECT DISTINCT ON (ts.practice_id, ts.student_id)
    p.id,
    p.name,
    ts.student_id,
    p.status AS practice_status,
    ts.status AS practice_student_status,
    COALESCE(p.allowed_attempts, 0) AS allowed_attempts,
    COALESCE(tp.level, '00'::character varying) AS difficulty,
    vep.question_count,
    COALESCE(vs.wrong_count, 0::bigint) AS wrong_count,
    vs.total_score,
    COALESCE((SELECT max(vs2.total_score)
              FROM v_student_practice_total_score vs2
              WHERE vs2.practice_id = p.id AND vs2.student_id = ts.student_id), 0::double precision) AS highest_score,
    COALESCE(psub_max.attempt_count, 0) AS attempt_count,
    COALESCE(lus.submission_id, 0) AS latest_unsubmitted_id,
    COALESCE(ls.submission_id, 0) AS latest_submitted_id,
    COALESCE(pm.submission_id, 0) AS pending_mark_id,
    ts.create_time,
    p.type,
    p.paper_id,
    COALESCE(tp.name , '') as paper_name,
    vep.total_score AS paper_total_score,
    p.exam_paper_id AS exam_paper_id,
    tp.suggested_duration AS suggested_duration
FROM t_practice_student ts
JOIN t_practice p ON ts.practice_id = p.id
JOIN t_paper tp ON p.paper_id = tp.id
JOIN v_exam_paper vep ON p.exam_paper_id = vep.id
LEFT JOIN (
    SELECT student_id, practice_id, MAX(attempt) AS attempt_count
    FROM t_practice_submissions
    GROUP BY student_id, practice_id
) psub_max ON psub_max.student_id = ts.student_id AND psub_max.practice_id = ts.practice_id
LEFT JOIN v_y_max_submitted_view vs ON vs.practice_id = p.id AND vs.student_id = ts.student_id
LEFT JOIN v_latest_unsubmitted_practice lus ON lus.practice_id = ts.practice_id AND lus.student_id = ts.student_id
LEFT JOIN v_latest_submitted_practice ls ON ls.practice_id = ts.practice_id AND ls.student_id = ts.student_id
LEFT JOIN v_latest_pending_mark_practice pm ON pm.practice_id = ts.practice_id AND pm.student_id = ts.student_id
ORDER BY ts.practice_id, ts.student_id, ts.create_time DESC;

comment on view v_z_practice_summary is
'v_z_practice_summary';

drop table if exists t_v_practice_summary;
create table t_v_practice_summary as select * from v_z_practice_summary;

/*==============================================================*/
/* View: v_z_practice_wrong_collection                          */
/*==============================================================*/
create or replace view v_z_practice_wrong_collection as
WITH 
latest_submission AS(
    SELECT 
        v.id AS practice_submission_id,
        v.student_id,
        v.practice_id,
        tps.wrong_attempt,
        tps.exam_paper_id
     FROM v_y_max_submitted_view v
     JOIN t_practice_submissions tps ON v.id = tps.id
     WHERE tps.status = '08'
),
wrong_questions AS (
    SELECT 
        tsa.question_id
    FROM t_student_answers tsa
    JOIN latest_submission ls
        ON tsa.practice_submission_id = ls.practice_submission_id 
        AND tsa.wrong_attempt = ls.wrong_attempt
    JOIN t_exam_paper_question tepq ON tsa.question_id = tepq.id
    WHERE tsa.answer_score < tepq.score
),
question_agg AS (
    SELECT 
        tepq.group_id,
        jsonb_agg(
            jsonb_build_object(
                'id', tepq.id,
                'type', tepq.type,
                'content', tepq.content,
                'options', tepq.options,
                'answers', tepq.answers,
                'score', tepq.score, 
                'analysis', tepq.analysis,
                'title', tepq.title,
                'answer_file_path', tepq.answer_file_path,
                'test_file_path', tepq.test_file_path,
                'input', tepq.input,
                'output', tepq.output,
                'example', tepq.example,
                'repo', tepq.repo,
                'order', tepq."order", 
                'group_id', tepq.group_id,
                'status', tepq.status,
                'question_attachments_path', tepq.question_attachments_path
            ) ORDER BY tepq."order"
        ) AS questions,
        SUM(tepq.score) AS group_total_score,
        COUNT(tepq.id) AS group_question_count
    FROM t_exam_paper_question tepq
    JOIN wrong_questions wq ON tepq.id = wq.question_id
    WHERE tepq.status = '00'
    GROUP BY tepq.group_id
),
group_data AS (
    SELECT 
        pg.id,
        pg.name,
        pg."order",
        pg.creator,
        pg.create_time,
        pg.updated_by,
        pg.update_time,
        pg.status,
        pg.addi,
        pg.exam_paper_id,
        COALESCE(qa.questions, '[]'::jsonb) AS questions,
        COALESCE(qa.group_total_score, 0) AS group_total_score,
        COALESCE(qa.group_question_count, 0) AS group_question_count
    FROM t_exam_paper_group pg
    JOIN question_agg qa ON qa.group_id = pg.id
    WHERE pg.status != '02'
),
paper_groups AS (
    SELECT 
        exam_paper_id,
        jsonb_agg(
            jsonb_build_object(
                'id', id,
                'name', name,
                'order', "order",
                'creator', creator,
                'create_time', create_time,
                'updated_by', updated_by,
                'update_time', update_time,
                'status', status,
                'addi', addi,
                'questions', questions
            ) ORDER BY "order"
        ) AS groups_data,
        SUM(group_total_score) AS total_score,
        SUM(group_question_count) AS question_count,
        COUNT(*) AS group_count
    FROM group_data
    WHERE questions <> '[]'::jsonb
    GROUP BY exam_paper_id
)
SELECT 	
    p.id,
    p.name,
    ls.student_id,
    ls.practice_id,
    ls.practice_submission_id AS practice_submission_id,
    p.creator,
    p.create_time,
    p.updated_by,
    p.update_time,
    p.status,
    COALESCE(pgrp.total_score, 0) AS total_score,
    COALESCE(pgrp.question_count, 0) AS question_count,
    COALESCE(pgrp.group_count, 0) AS group_count,
    COALESCE(pgrp.groups_data, '[]'::jsonb) AS groups_data
FROM t_exam_paper p 
JOIN paper_groups pgrp ON pgrp.exam_paper_id = p.id
JOIN latest_submission ls ON ls.exam_paper_id = p.id
WHERE p.status = '00';

comment on view v_z_practice_wrong_collection is
'学生某次练习提交错题集视图';

drop table if exists t_v_practice_wrong_collection;

create table t_v_practice_wrong_collection as select * from v_z_practice_wrong_collection;

alter table t_domain_api
   add constraint FK_ACT_REF_DOMAIN foreign key (domain)
      references t_domain (id)
      on delete restrict on update restrict;

alter table t_domain_api
   add constraint FK_ACT_REF_API foreign key (api)
      references t_api (id)
      on delete restrict on update restrict;

alter table t_exam_paper_group
   add constraint FK_exam_group_paper foreign key (exam_paper_id)
      references t_exam_paper (id)
      on delete cascade;

alter table t_exam_paper_question
   add constraint FK_exam_question_group foreign key (group_id)
      references t_exam_paper_group (id)
      on delete cascade;

alter table t_exam_session
   add constraint fk_t_exam_session_exam_info foreign key (exam_id)
      references t_exam_info (id)
      on delete cascade;

alter table t_examinee
   add constraint fk_t_examinee_exam_session foreign key (exam_session_id)
      references t_exam_session (id)
      on delete cascade;

alter table t_insure_attach
   add constraint FK_T_INSURE_REF_T_USER foreign key (t_u_id)
      references t_user (id)
      on delete restrict on update restrict;

alter table t_paper_group
   add constraint FK_PAPER_GROUP_T_PAPER foreign key (paper_id)
      references t_paper (id)
      on delete cascade on update cascade;

alter table t_paper_question
   add constraint FK_T_PAPER__FK_PAPER__T_QUESTI foreign key (bank_question_id)
      references t_question (id)
      on delete restrict on update restrict;

alter table t_paper_question
   add constraint FK_PAPER_QUESTION_PAPER foreign key (group_id)
      references t_paper_group (id)
      on delete cascade on update cascade;

alter table t_question
   add constraint FK_question_question_bank foreign key (belong_to)
      references t_question_bank (id)
      on delete set null on update restrict;

alter table t_student_answers
   add constraint FK_exam_answer_question foreign key (question_id)
      references t_exam_paper_question (id)
      on delete cascade;

alter table t_user_domain
   add constraint fk_user_domain_uid_REF_USER_id foreign key (sys_user)
      references t_user (id)
      on delete restrict on update restrict;

alter table t_user_domain
   add constraint FK_USER_domain_REF_DOMAIN foreign key (domain)
      references t_domain (id)
      on delete restrict on update restrict;

alter table t_wx_user
   add constraint FK_WX_USER_REF_USER foreign key (id)
      references t_user (id)
      on delete restrict on update restrict;

alter table t_xkb_user
   add constraint FK_T_XKB_US_REFERENCE_T_USER foreign key (id)
      references t_user (id)
      on delete cascade on update cascade;

