  * **Table of contents**
    * [What is an message?](Messages#What_is_an_message?.md)
    * [Different types](Messages#Different_types.md)
      * [Request Message type](Messages#Request_Message_type.md)
      * [Response Message type](Messages#Response_Message_type.md)
    * [Parsing](Messages#Parsing.md)
    * [See also](Messages#See_also.md)

# What is an message? #
An message is an single chunk of data which is sent over either HTTP or WebSocket protocols. Each message can be one of the predefined different types and each type can be easily parsed.

Messages are strictly _encoded in JSON_.

# Different types #
There are an few different types of messages inside the inner workings of Organics.

### Request Message type ###
  * This Message type is sent when you wish to make an request to the other connection side (E.g. an request from client to server, or server to client).
  * In JSON this message looks like `[id, requestName, args]`.
    * The id value would be any JSON Number.
      * Any value _other than an -1_, will be forwarded back along with the Response message type, when it is sent. This can be used to uniquely keep track of an Request's corresponding Response message.
      * An value of -1, means that "This side of the connection (client or server) does not wish for an Response message to be sent corresponding to this Request message.
    * The requestName value would be any JSON data type, it should be any unique value which represents this Request to be handled.
    * The args value is an JSON Array type of any number of any JSON data types, which are arguments that go with the Request.
      * If the args value would otherwise be an empty JSON array, `"[]"` then it can be omitted for bandwidth reasons.

### Response Message type ###
  * This Message type is sent when your Request's associated handler function returns any data types.
  * This message is not sent if the id value associated with this Response's Request Message, is -1.
  * In JSON this message looks like `[id, args]`
    * The id value is any JSON Number, it _is_ the id value that was originally sent with this Response's associated Request Message.
    * The args value is an JSON Array type of any number of any JSON data types, which are arguments that go with the Response.

# Parsing #
In order to parse any Organics Messages, you will need an JSON decoder of some sort, as all messages are encoded in JSON.

This section describes how you can determine the Message's type, after it has been decoded.

  * Pay close attention to the following Messages and their lengths:
| Length | JSON                          | Type     |
|:-------|:------------------------------|:---------|
| Length | JSON                          | Type     |
| 3      | `[id, requestName, args]`     | Request  |
| 2      | `[id, args]`                  | Response |
| 1      | `[id]`                        | Response |

  1. Decode the JSON message into an JSON Array type.
  1. Let 'data' represent the decoded JSON Array.
  1. Let 'dataLength' represent the decoded JSON Array's length.
  1. `if dataLength == 3:`
    * Message is Request type, in format of `[id, requestName, args]`.
  1. `if dataLength == 2:`
    * Message is Response type, in format of `[id, args]`.
  1. If dataLength == 1 then
    * Message is Request type, in format of `[id]`

# See also #
While strictly not Messages, another portion of the protocol Organics follows is pings. (See [Security](Security#Pings.md))