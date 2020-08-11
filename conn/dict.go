package conn

const MAX_MESSAGE_LEN = 64 * 1024 * 1024

const CHAN = "_chan"

// channel
const CHANS = "_chas"
const AUTH = "_auth"
const ATTACH = "_atta"
const DETACH = "_deta"
const PING = "_ping"
const QUERY = "_quer"
const MINE = "_mine"
const CUSTOM = "_cust"
const CTRL = "_ctrl"

// params
const SOCKET_ID = "_soid"
const SLOT_ID = "_slid"
const VALUE = "_valu"
const LABEL = "_labe"
const TO = "_to"
const FROM = "_from"
const SHARE = "_shar"
const ROOT = "_root"
const ATTR = "_attr"
const ADDR = "_addr"

// message id
const ID = "_id"

// error
const CODE = "_code"
const ERROR = "_erro"

// slot event channel
const SLOT_READY = "_slre"
const SLOT_BREAK = "_slbr"
const SLOT_ATTACH = "_slat"
const SLOT_DETACH = "_slde"
const SLOT_KILL = "_slki"
const SLOT_SEND = "_slse"
const SLOT_RECV = "_slrc"

// attr
const SEND_NUM = "_snum"
const RECV_NUM = "_rnum"

// crypto
const AES_128_GCM = "AES_128_GCM"
const AES_256_GCM = "AES_256_GCM"
const CHACHA20_POLY1305 = "CHACHA20_POLY1305"

// network
const HAND = "_hand"
const METHOD = "_meth"
const SECURE = "_secu"
const ORIGIN = "_orig"

const ACCESS = "_acce"
const SECRET = "_secr"
