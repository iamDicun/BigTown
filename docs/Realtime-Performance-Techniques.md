# Kỹ thuật xử lý Realtime & Hiệu năng

Tài liệu này tổng hợp các **kỹ thuật/nguyên lý chung** đã dùng để xử lý các nhóm vấn đề gặp phải khi
đưa một hệ thống realtime (nhiều người chơi, đồng bộ vị trí, va chạm) từ local ra production. Mục
đích là để hiểu *bản chất vấn đề* và *cách tư duy giải quyết*, có thể áp dụng lại cho các vấn đề
tương tự sau này, không phải nhật ký "đã sửa file nào".

---

## 1. Race condition trong "kiểm tra rồi mới làm" (check-then-act)

**Vấn đề chung**: Khi nhiều luồng/kết nối cùng đọc một trạng thái dùng chung để quyết định hành
động ("nếu chưa tồn tại thì tạo mới"), nếu bước *đọc* và bước *ghi* không nằm trong cùng một
đơn vị khoá (lock), thì 2 luồng có thể cùng đọc thấy "chưa tồn tại" và cùng thực hiện hành động
tưởng là duy nhất — dẫn tới trùng lặp hoặc mất đồng bộ. Đây là một trong những lỗi race condition
phổ biến nhất khi nhiều kết nối/tab của cùng một người dùng thao tác gần như đồng thời.

**Cách xử lý**: Gộp toàn bộ chuỗi "kiểm tra + quyết định + ghi" vào **cùng một lock**, không tách
"đọc trạng thái" ra một bước riêng ở tầng gọi rồi mới "ghi" ở một bước khác. Nơi giữ trạng thái
(data store) phải là nơi duy nhất quyết định "đây có phải lần đầu hay không", vì chỉ nó mới thấy
được bức tranh nhất quán tại đúng thời điểm ghi.

**Khi nào áp dụng**: Bất kỳ luồng "join/register lần đầu", "tạo nếu chưa có", đếm số lượng, hay bất
kỳ logic nào phụ thuộc vào "trạng thái hiện có" để quyết định nhánh xử lý, trong môi trường có thể
có nhiều request/kết nối chạy song song trên cùng một entity.

---

## 2. Vật lý "tween" (animation di chuyển) và physics body xung đột nhau

**Vấn đề chung**: Trong các engine game 2D, một đối tượng vừa được điều khiển bằng tween (nội suy
mượt vị trí theo thời gian, ghi đè trực tiếp toạ độ) vừa gắn một physics body "động" (dynamic body,
kể cả khi đánh dấu bất động — immovable) sẽ bị giật/rung liên tục. Lý do: physics engine tự đồng bộ
lại vị trí body từ transform mỗi khung hình rồi tính toán vật lý dựa trên đó, tạo ra xung đột hai
chiều với việc tween liên tục ghi đè toạ độ.

**Cách xử lý**: Tách rời hoàn toàn hai vai trò — *hiển thị* (sprite tween, không có physics body) và
*va chạm* (một đối tượng vô hình khác dùng physics body **tĩnh** — static body, không tham gia
bước tính toán/đồng bộ mỗi khung hình của engine). Vị trí của đối tượng va chạm được "dán" (snap)
theo vị trí xác thực mới nhất mỗi khi cập nhật, không tween theo.

**Khi nào áp dụng**: Bất kỳ đối tượng nào vừa cần di chuyển mượt bằng animation/tween, vừa cần chặn
va chạm với đối tượng khác, trong một physics engine tách biệt static/dynamic body.

---

## 3. Mô hình di chuyển "server quyết định, client đoán trước" (server-authoritative movement)

**Vấn đề chung**: Nếu client gửi vị trí lên và chờ server xác nhận rồi mới hiển thị, độ trễ mạng sẽ
khiến nhân vật di chuyển giật cục, cảm giác không "real". Ngược lại nếu client tự quyết định vị trí
hoàn toàn, server không thể chống gian lận/đảm bảo tính nhất quán giữa nhiều người chơi.

**Cách xử lý**: Client hiển thị ngay lập tức theo input của người chơi (dự đoán lạc quan), đồng thời
gửi báo cáo vị trí lên server theo nhịp cố định (không phải mỗi frame, để tránh spam mạng). Server
là nơi duy nhất quyết định vị trí "chính thức" — validate tốc độ, biên bản đồ, va chạm với người
khác — rồi phát (broadcast) vị trí đã được chấp nhận cho tất cả người chơi khác. Nếu client gửi lên
một vị trí không hợp lệ, server không broadcast mà gửi riêng một "lệnh sửa" (correction) về đúng
client đó để nó tự chỉnh lại, không ảnh hưởng người khác.

**Đánh đổi cần biết**: Cơ chế này không loại bỏ được độ trễ — nó chỉ chuyển độ trễ từ "giật khi tự
di chuyển" sang "thấy người khác di chuyển hơi trễ hơn thực tế một chút" (bằng khoảng round-trip
mạng). Khi độ trễ mạng tăng, tần suất bị "sửa lại" (correction) cũng tăng theo — vì view của mỗi
client về vị trí người khác càng lúc càng cũ hơn thực tế, không phải vì server "mất khả năng"
validate.

**Khi nào áp dụng**: Game nhiều người chơi thời gian thực, nơi cảm giác phản hồi tức thời của chính
người chơi quan trọng hơn độ chính xác tuyệt đối của vị trí người khác tại mọi thời điểm.

---

## 4. Đưa các phép tính "tĩnh"/hiếm đổi ra khỏi đường xử lý tần suất cao (hot path)

**Vấn đề chung**: Một thao tác lặp lại rất nhiều lần mỗi giây (ví dụ: xử lý mỗi gói tin di chuyển)
nếu bên trong nó gọi đến một nguồn dữ liệu chậm (database, API ngoài...) để lấy thông tin *thực ra
gần như không đổi* giữa các lần gọi, thì chi phí của nguồn dữ liệu chậm đó sẽ bị nhân lên theo tần
suất gọi. Trên máy local, chi phí này gần như 0 nên không ai nhận ra; nhưng khi triển khai thật (dữ
liệu, mạng ở xa nhau, hoặc dịch vụ dùng chung bị tải), chi phí này lộ rõ và làm nghẽn toàn bộ luồng
xử lý — biểu hiện ra ngoài là "phản hồi chậm dần", dễ bị nhầm là do băng thông/độ trễ mạng thuần
tuý trong khi bản chất là do số lượt gọi ra nguồn dữ liệu chậm.

**Cách xử lý**: Xác định trong đường xử lý tần suất cao, phần dữ liệu nào là **tra cứu lặp lại**
(cùng một câu hỏi, cùng một câu trả lời trong suốt một phiên) và phần nào là **thay đổi thực sự**
mỗi lần gọi. Với phần tra cứu lặp lại: lưu sẵn kết quả trong bộ nhớ (cache) sau lần lấy đầu tiên,
hoặc tính toán một lần rồi giữ nguyên trong suốt vòng đời của phiên làm việc thay vì hỏi lại nguồn
chậm mỗi lần. Chỉ giữ lại việc gọi ra nguồn chậm ở những thời điểm ít xảy ra (ví dụ: lúc bắt đầu
phiên), không phải ở mỗi nhịp lặp.

**Khi nào áp dụng**: Bất kỳ vòng lặp/handler nào chạy với tần suất cao (mỗi giây nhiều lần) mà bên
trong có gọi ra một nguồn dữ liệu có độ trễ không đảm bảo (DB, service khác) để lấy thông tin có
bản chất tĩnh hoặc đã biết trước trong phạm vi phiên hiện tại.

---

## 5. Chỉ mục phụ trong bộ nhớ để tránh tra cứu chéo qua dịch vụ khác

**Vấn đề chung**: Một thao tác cần "đổi" từ một định danh (ví dụ: định danh người dùng đã xác thực)
sang một định danh khác (ví dụ: định danh đối tượng trong game) để tra cứu trạng thái. Nếu phép đổi
này luôn phải hỏi qua một dịch vụ/nguồn dữ liệu khác thì nó lặp lại vấn đề ở mục 4, đặc biệt khi
phép đổi này xảy ra trên đường xử lý tần suất cao.

**Cách xử lý**: Nếu mối quan hệ giữa hai định danh này đã được xác lập một lần (ví dụ: lúc bắt đầu
phiên) và không đổi trong suốt phiên, hãy lưu thêm một **chỉ mục phụ** (secondary index) ngay trong
cấu trúc dữ liệu đang giữ trạng thái trong bộ nhớ, ánh xạ trực tiếp định danh này sang định danh kia.
Các lần tra cứu sau chỉ cần đọc chỉ mục này (tốc độ tra cứu tức thời), không cần hỏi lại dịch vụ
khác nữa. Chỉ mục phải được dọn dẹp đúng lúc đối tượng không còn tồn tại nữa để tránh rò rỉ bộ nhớ
và dữ liệu treo (stale).

**Khi nào áp dụng**: Khi một hệ thống có nhiều tầng định danh cho cùng một thực thể (người dùng ↔
nhân vật, phiên ↔ tài khoản...) và việc quy đổi giữa các tầng này xảy ra thường xuyên hơn tần suất
mà dữ liệu nguồn thực sự thay đổi.

---

## 6. Gộp các lệnh gọi độc lập để chạy song song thay vì nối tiếp

**Vấn đề chung**: Khi khởi tạo một màn hình/tính năng cần lấy dữ liệu từ nhiều nguồn độc lập (không
nguồn nào phụ thuộc kết quả của nguồn kia), nếu gọi tuần tự (đợi xong cái này mới gọi cái tiếp theo)
thì tổng thời gian chờ là **tổng** thời gian của từng lệnh gọi, dù chúng có thể chạy đồng thời.

**Cách xử lý**: Xác định đúng những lệnh gọi nào thực sự độc lập (không lệnh nào cần input là kết
quả của lệnh khác), sau đó phát hết ra cùng lúc và chờ tất cả cùng hoàn tất. Tổng thời gian chờ khi
đó chỉ còn bằng lệnh gọi **chậm nhất**, không phải tổng cộng.

**Khi nào áp dụng**: Màn hình khởi tạo/tải dữ liệu ban đầu gọi nhiều API khác nhau, đặc biệt quan
trọng khi độ trễ mỗi lệnh gọi lớn (dịch vụ ở xa, mới khởi động lại...).

---

## 7. Cookie giữa các miền khác nhau (cross-site) so với cùng miền (same-site)

**Vấn đề chung**: Trình duyệt quyết định có gửi kèm cookie trong một request nền (không phải điều
hướng trang) hay không dựa trên việc request đó có được xem là "cùng site" với trang đang mở hay
không — "site" được tính theo miền gốc + giao thức, **không tính theo cổng (port)**. Khi frontend
và backend được triển khai trên hai miền thực sự khác nhau (khác domain gốc), các thuộc tính cookie
mặc định phù hợp cho môi trường phát triển local (cùng miền, khác cổng) sẽ không còn hoạt động —
cookie bị trình duyệt âm thầm không gửi kèm, biểu hiện ra là các request cần xác thực bằng cookie
luôn thất bại như thể cookie chưa từng tồn tại.

**Cách xử lý**: Hai bộ cấu hình khác nhau cho hai bối cảnh khác nhau, không phải một bộ "chuẩn" áp
dụng chung. Môi trường cùng site (local) giữ cấu hình lỏng hơn và không yêu cầu kết nối mã hoá.
Môi trường khác site (production) bắt buộc nới quyền gửi cookie qua nhiều site hơn, và quy tắc của
trình duyệt là nếu nới quyền này thì bắt buộc phải đi kèm yêu cầu kết nối phải được mã hoá — thiếu
một trong hai vế thì trình duyệt từ chối toàn bộ cookie.

**Khi nào áp dụng**: Bất kỳ hệ thống nào tách frontend/backend ra hai domain khác nhau khi triển
khai thật nhưng chạy chung một domain (chỉ khác cổng) lúc phát triển local.

---

## 8. Xử lý lỗi kiểu "cố gắng tốt nhất" (best-effort) cho thao tác không quan trọng bằng việc dọn dẹp cục bộ

**Vấn đề chung**: Một thao tác có hai phần: báo cho hệ thống ở xa biết (ví dụ: thông báo cho server)
và dọn dẹp trạng thái ngay tại chỗ (ví dụ: xoá phiên đăng nhập trên máy người dùng). Nếu code viết
theo kiểu "phải báo thành công cho server trước, rồi mới dọn dẹp cục bộ", thì bất kỳ lý do gì khiến
phần báo cho server thất bại (mất mạng, cấu hình sai, hết hạn...) sẽ chặn đứng luôn phần dọn dẹp cục
bộ — người dùng rơi vào trạng thái nửa vời: tưởng đã thoát nhưng thực chất trạng thái cũ vẫn còn
sống.

**Cách xử lý**: Với các hành động mà phần "dọn dẹp cục bộ" là điều **bắt buộc phải xảy ra** để trải
nghiệm người dùng đúng đắn (ví dụ: thoát khỏi phiên làm việc), tách rõ: phần báo cho server là
"cố gắng tốt nhất, không chặn" (thử gọi, nhưng dù thành công hay thất bại đều không ảnh hưởng bước
sau), còn phần dọn dẹp cục bộ luôn luôn chạy bất kể kết quả phần trên.

**Khi nào áp dụng**: Các hành động "huỷ/thoát/xoá" nơi hậu quả của việc *không* dọn dẹp cục bộ
(người dùng vẫn ở trạng thái cũ) nghiêm trọng hơn hậu quả của việc server không kịp biết ngay lập
tức.
