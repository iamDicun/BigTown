Chia làm 4 nhóm, từ "phải pass trước khi merge" đến "đo để biết đã tối ưu chưa". Mình nêu cả *test cái gì* lẫn *làm thế nào*.

## 1. Bốn bài đúng đắn cốt lõi (bắt buộc, tự động hoá được)

Đây là 4 kịch bản trong checklist — quan trọng nhất, và đều test được bằng Go test đánh thẳng vào usecase/handler với DB thật (dùng `testcontainers-go` hoặc một Postgres docker riêng cho test).

**Overspend / coin âm.** Set 1 character coin đúng bằng giá 1 món. Bắn 50 request place song song (dùng `sync.WaitGroup` + goroutine gọi handler). Sau đó `SELECT coins` — phải ≥ 0, và đúng **1** placement được tạo. Đây là bài quan trọng nhất vì nó là lỗ double-spend. Ở Tier 1, không có guard `WHERE coins >= price` là bài này fail ngay.

**Đặt trùng ô.** Hai request place cùng `(map_id, x, y)` chạy đồng thời → đúng 1 thành công, cái kia trả lỗi "ô đã có vật thể" sạch (không phải 500). Đếm số row trong `map_placements` cho ô đó = 1.

**Double-delete.** Xoá cùng một `placementId` hai lần liên tiếp (hoặc song song) → coin chỉ được hoàn **một lần**. Kiểm `coins` cuối và số dòng `reward_events` type `decoration_refund` = 1.

**Crash / durability.** Đặt vài món rồi kill process (SIGKILL, không phải graceful) → restart → `SELECT` phải khớp với những gì client đã nhận. Tier 1: phải khớp 100% (đồng bộ). Tier 2: chấp nhận mất ≤ flush-interval, nhưng **không được** có vật thể ma (thứ đã broadcast mà DB không có sau khi đã flush ổn định).

Cách chạy phần song song: `go test -race` — race detector sẽ bắt data race trong actor/wallet nếu có. Chạy luôn `-count=20` để lộ lỗi phụ thuộc timing.

## 2. Realtime đa client (thủ công + script)

Cái này khó cover bằng unit test, nên làm hai lớp:

**Thủ công 2 tab.** Mở 2 trình duyệt/tab đăng nhập 2 account, cùng map. Tab A đặt vật thể → tab B phải thấy **ngay** không cần refresh; A xoá → B mất ngay. Kiểm cả chiều ngược lại. Đây là smoke test bạn nên làm mỗi lần đụng vào realtime.

**Kiểm echo.** Ở tab A, sau khi đặt, vật thể **không** được nhân đôi (client đặt vừa dùng response API, vừa nhận broadcast của chính mình — phần lọc `character_id === gameStore.characterId` phải chặn). Đặt 5 món liên tiếp nhanh ở A, đếm sprite trên cả A và B phải bằng nhau và bằng 5.

**Reload = nguồn chân lý.** Sau một loạt thao tác ở nhiều tab, F5 cả hai → state hiển thị phải giống hệt nhau và giống DB. Đây là bài phát hiện "coin/placement optimistic bị lệch".

Nếu muốn tự động: viết một script Node dùng client `centrifuge` mở N kết nối, subscribe channel `room:<mapCode>`, cho vài "client" spam place qua HTTP, rồi assert mọi client nhận đủ N event `decoration_placed` đúng thứ tự.

## 3. Test đặc thù Tier 2 (nếu đã lên actor)

**Bất biến ví.** Đây là chỗ dễ desync nhất. Trong lúc 1 player đang resident trong map actor, kích một nguồn coin khác (phần thưởng combat/leaderboard nếu có) → sau đó player đặt vật thể → coin phải phản ánh cả hai. Nếu nguồn kia còn ghi thẳng `characters.coins` (không qua command), bài này sẽ lộ ra coin bị "nuốt" khi actor flush đè giá trị tuyệt đối. Test này chính là để chứng minh PR4 làm đúng.

**Ordering.** Bắn place → delete → place cùng ô rất nhanh. State cuối phải nhất quán (ô có vật thể), không có kiểu delete tới sau khi place-2 rồi ô lại trống.

**Shutdown drain.** Gọi `RoomManager.Shutdown()` khi còn op trong queue → `SELECT` phải có đủ, không mất.

## 4. Hiệu năng / độ trễ (để trả lời "đã tối ưu chưa")

Đây là lý do bạn làm Tier 2, nên phải đo, nếu không thì không biết có đáng công không.

Dùng **k6**, **vegeta**, hoặc **bombardier** bắn vào endpoint `/editor/place`. Đo **p50/p95/p99 latency** và **throughput (req/s)** ở các mức đồng thời 10 / 50 / 200 client.

So sánh trực tiếp Tier 1 vs Tier 2 trên cùng máy, cùng data:
- Tier 1: latency sẽ tăng dần theo tải (vì mỗi request chờ commit + lock contention trên `characters`/`map_placements`). Quan sát p99 leo thang khi tăng đồng thời.
- Tier 2: latency phải gần **phẳng** theo tải (RAM, không chạm DB trên hot path). Nếu Tier 2 *không* phẳng hơn Tier 1 → actor đang bị chặn ở đâu đó (khả năng cao là bạn lỡ publish Centrifuge hoặc gọi DB *bên trong* vòng mutate thay vì offload).

Kèm theo, log các metric ở mục 9 của tài liệu Tier 2: độ sâu queue `cmds`/`dirty`, thời gian flush, số op/flush. Nếu queue depth tăng dần không về 0 → hot path vẫn còn nghẽn hoặc writer flush chậm hơn tốc độ nạp.

---

Thứ tự thực tế: chạy nhóm 1 (`go test -race -count=20`) mỗi lần build — đây là hàng rào chặn regression. Nhóm 2 làm thủ công 2 tab sau mỗi thay đổi realtime. Nhóm 3 chỉ khi lên Tier 2. Nhóm 4 chạy một lần để xác nhận Tier 2 thật sự phẳng hơn, rồi thỉnh thoảng đo lại.

Một mẹo: bài số 1 (overspend) và bài bất biến ví là hai bài dễ *tưởng pass* nhất — nếu không bắn đủ song song, chúng vẫn xanh. Nên luôn ≥ 50 goroutine và `-race`, đừng test tuần tự rồi yên tâm.

Muốn mình viết sẵn file test Go cho 4 bài nhóm 1 (dạng table-driven, dùng testcontainers) để bạn chỉ việc điền tên hàm thật vào không?