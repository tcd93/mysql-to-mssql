_For beginner's introduction to binlog, see [link](https://dev.mysql.com/doc/internals/en/binary-log-overview.html#:~:text=The%20binary%20log%20is%20a,was%20introduced%20in%20MySQL%203.23.)_

### __This page only covers ROW-based logging mode__

# MySQL Binlog
[sql/log_event.h](https://github.com/mysql/mysql-server/blob/ee4455a33b10f1b1886044322e4893f587b319ed/sql/log_event.h#L166-L172)
> This log consists of events.  Each event has a fixed-length header,
> possibly followed by a variable length data body.
> 
> The data body consists of an optional fixed length segment (post-header)
> and  an optional variable length segment.
> 
> All numbers, whether they are 16-, 24-, 32-, or 64-bit numbers,
> are stored in little endian, i.e., the least significant byte first,
> unless otherwise specified.

## [Header Structure](https://github.com/mysql/mysql-server/blob/ee4455a33b10f1b1886044322e4893f587b319ed/sql/log_event.h#L738-L741)
    +---------+---------+---------+------------+-----------+-------+
    |timestamp|type code|server_id|event_length|end_log_pos|flags  |
    |4 bytes  |1 byte   |4 bytes  |4 bytes     |4 bytes    |2 bytes|
    +---------+---------+---------+------------+-----------+-------+

_You can see the go-mysql's replication `Decode` function in [action](https://github.com/siddontang/go-mysql/blob/0c5789dd0bd378b4b84f99b320a2d35a80d8858f/replication/event.go#L70-L100)_

The _type code_ indicates event type, eg. row insert, update, delete...

A detailed list of supported codes can be found in [libbinlogevents/include/binlog_event.h](https://github.com/mysql/mysql-server/blob/ee4455a33b10f1b1886044322e4893f587b319ed/libbinlogevents/include/binlog_event.h#L265-L355)
```
enum Log_event_type {

  UNKNOWN_EVENT = 0,

  START_EVENT_V3 = 1,
  QUERY_EVENT = 2,
  STOP_EVENT = 3,
  ROTATE_EVENT = 4,
  INTVAR_EVENT = 5,

  SLAVE_EVENT = 7,

  APPEND_BLOCK_EVENT = 9,
  DELETE_FILE_EVENT = 11,

  RAND_EVENT = 13,
  USER_VAR_EVENT = 14,
  FORMAT_DESCRIPTION_EVENT = 15,
  XID_EVENT = 16,
  BEGIN_LOAD_QUERY_EVENT = 17,
  EXECUTE_LOAD_QUERY_EVENT = 18,

  TABLE_MAP_EVENT = 19,

  /**
    The V1 event numbers are used from 5.1.16 until mysql-5.6.
  */
  WRITE_ROWS_EVENT_V1 = 23,
  UPDATE_ROWS_EVENT_V1 = 24,
  DELETE_ROWS_EVENT_V1 = 25,

  /**
    Something out of the ordinary happened on the master
   */
  INCIDENT_EVENT = 26,

  /**
    Heartbeat event to be send by master at its idle time
    to ensure master's online status to slave
  */
  HEARTBEAT_LOG_EVENT = 27,

  /**
    In some situations, it is necessary to send over ignorable
    data to the slave: data that a slave can handle in case there
    is code for handling it, but which can be ignored if it is not
    recognized.
  */
  IGNORABLE_LOG_EVENT = 28,
  ROWS_QUERY_LOG_EVENT = 29,

  /** Version 2 of the Row events */
  WRITE_ROWS_EVENT = 30,
  UPDATE_ROWS_EVENT = 31,
  DELETE_ROWS_EVENT = 32,

  GTID_LOG_EVENT = 33,
  ANONYMOUS_GTID_LOG_EVENT = 34,

  PREVIOUS_GTIDS_LOG_EVENT = 35,

  TRANSACTION_CONTEXT_EVENT = 36,

  VIEW_CHANGE_EVENT = 37,

  /* Prepared XA transaction terminal event similar to Xid */
  XA_PREPARE_LOG_EVENT = 38,

  /**
    Extension of UPDATE_ROWS_EVENT, allowing partial values according
    to binlog_row_value_options.
  */
  PARTIAL_UPDATE_ROWS_EVENT = 39,

  TRANSACTION_PAYLOAD_EVENT = 40,

  /**
    Add new events here - right above this comment!
    Existing events (except ENUM_END_EVENT) should never change their numbers
  */
  ENUM_END_EVENT /* end marker */
};
```

## Body Structure (for ROWS EVENT only)
  The Post-Header has the following components:
  <table>
  <tr>
    <th>Name</th>
    <th>Format</th>
    <th>Description</th>
  </tr>
  <tr>
    <td>table_id</td>
    <td>6 bytes unsigned integer</td>
    <td>The number that identifies the table</td>
  </tr>
  <tr>
    <td>flags</td>
    <td>2 byte bitfield</td>
    <td>Reserved for future use; currently always 0.</td>
  </tr>
  </table>

  The Body has the following components:
  <table>
  <tr>
    <th>Name</th>
    <th>Format</th>
    <th>Description</th>
  </tr>
  <tr>
    <td>width</td>
    <td>packed integer</td>
    <td>Represents the number of columns in the table</td>
  </tr>
  <tr>
    <td>cols</td>
    <td>Bitfield, variable sized</td>
    <td><p>Indicates whether each column is used, one bit per column.</p>
        For this field, the amount of storage required is
        INT((width + 7) / 8) bytes. </td>
  </tr>
  <tr>
    <td>extra_row_info</td>
    <td>An object of class Extra_row_info</td>
    <td><p>The class Extra_row_info will be storing the information related
        to m_extra_row_ndb_info and partition info (partition_id and
        source_partition_id).</p>
        <p>At any given time a Rows_event can have both, one
        or none of ndb_info and partition_info present as part of Rows_event.</p>
        In case both ndb_info and partition_info are present then below will
        be the order in which they will be stored.
        <pre>
        +----------+--------------------------------------+
        |type_code |        extra_row_ndb_info            |
        +--- ------+--------------------------------------+
        | NDB      |Len of ndb_info |Format |ndb_data     |
        | 1 byte   |1 byte          |1 byte |len - 2 byte |
        +----------+----------------+-------+-------------+
        In case of INSERT/DELETE
        +-----------+----------------+
        | type_code | partition_info |
        +-----------+----------------+
        |   PART    |  partition_id  |
        | (1 byte)  |     2 byte     |
        +-----------+----------------+
        In case of UPDATE
        +-----------+------------------------------------+
        | type_code |        partition_info              |
        +-----------+--------------+---------------------+
        |   PART    | partition_id | source_partition_id |
        | (1 byte)  |    2 byte    |       2 byte        |
        +-----------+--------------+---------------------+
        source_partition_id is used only in the case of Update_event
        to log the partition_id of the source partition.
        </pre>
        This is the format for any information stored as extra_row_info.</br>
        type_code is not a part of the class Extra_row_info as it is a constant
        values used at the time of serializing and decoding the event.
   </td>
  </tr>
  <tr>
    <td>columns_before_image</td>
    <td>vector of elements of type uint8_t</td>
    <td><p>For DELETE and UPDATE only.</p>
        <p>Bit-field indicating whether each column is used
        one bit per column.</p><p>For this field, the amount of storage
        required for N columns is INT((N + 7) / 8) bytes.</p></td>
  </tr>
  <tr>
    <td>columns_after_image</td>
    <td>vector of elements of type uint8_t</td>
    <td><p>For WRITE and UPDATE only.</p>
        <p>Bit-field indicating whether each column is used in the
        UPDATE_ROWS_EVENT and WRITE_ROWS_EVENT after-image; one bit per column.</p>
        For this field, the amount of storage required for N columns
        is INT((N + 7) / 8) bytes.
        <pre>
          +-------------------------------------------------------+
          | Event Type | Cols_before_image | Cols_after_image     |
          +-------------------------------------------------------+
          |  DELETE    |   Deleted row     |    NULL              |
          |  INSERT    |   NULL            |    Inserted row      |
          |  UPDATE    |   Old     row     |    Updated row       |
          +-------------------------------------------------------+
        </pre>
    </td>
  </tr>
  <tr>
    <td>row</td>
    <td>vector of elements of type uint8_t</td>
    <td> <p>A sequence of zero or more rows. The end is determined by the size
         of the event. Each row has the following format:</p>
         <ul>
           <li> A Bit-field indicating whether each field in the row is NULL.
             Only columns that are "used" according to the second field in
             the variable data part are listed here. If the second field in
             the variable data part has N one-bits, the amount of storage
             required for this field is INT((N + 7) / 8) bytes. </li>
           <li> The row-image, containing values of all table fields. This only
             lists table fields that are used (according to the second field
             of the variable data part) and non-NULL (according to the
             previous field). In other words, the number of values listed here
             is equal to the number of zero bits in the previous field.
             (not counting padding bits in the last byte). </li>
             </ul>
             <pre>
                For example, if a INSERT statement inserts into 4 columns of a
                table, N= 4 (in the formula above).
                length of bitmask= (4 + 7) / 8 = 1
                Number of fields in the row= 4.
                        +------------------------------------------------+
                        |Null_bit_mask(4)|field-1|field-2|field-3|field 4|
                        +------------------------------------------------+
             </pre>
    </td>
  </tr>
  </table>

More documentation about size of types can be found [here](https://github.com/mysql/mysql-server/blob/ee4455a33b10f1b1886044322e4893f587b319ed/libbinlogevents/include/rows_event.h#L111)