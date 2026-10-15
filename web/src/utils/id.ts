// 任務短 ID：xid 末 6 碼（同一時段建立的任務在這段最不容易撞），用來分辨同名任務
export const shortId = (id: string) => `#${id.slice(-6)}`
