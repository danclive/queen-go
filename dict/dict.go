package dict

const MAX_MESSAGE_LEN = 64 * 1024 * 1024

const CHAN = "_ch"

// channel
const CHANS = "_cs"
const ATTACH = "_ah"
const DETACH = "_dh"
const PING = "_pi"
const QUERY = "_qu"
const MINE = "_mi"
const CUSTOM = "_cu"
const CTRL = "_ct"

// params
const SOCKET_ID = "_so"
const SLOT_ID = "_sl"
const VALUE = "_va"
const LABEL = "_la"
const TO = "_to"
const FROM = "_fr"
const SHARE = "_sh"
const ROOT = "_ro"
const ATTR = "_at"
const ADDR = "_ad"

// message id
const ID = "_id"

// error
const CODE = "_co"
const ERROR = "_er"

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
const AES_128_GCM = "A1G"
const AES_256_GCM = "A2G"
const CHACHA20_POLY1305 = "CP1"

// network
const HAND = "_ha"
const METHOD = "_me"
const SECURE = "_se"
const ORIGIN = "_or"

const ACCESS = "_acce"
const SECRET = "_secr"
