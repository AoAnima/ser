package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"math"
	"math/big"
	"net/smtp"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	. "aoanima.ru/ConnQuic"
	. "aoanima.ru/DataBase"
	. "aoanima.ru/Logger"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	json "github.com/json-iterator/go"
	"github.com/quic-go/quic-go"
)

var клиент = make(Клиент)
var Сервис ИмяСервиса = "Авторизация"

// TODO
var БазаДанных Таблица

func main() {
	Инфо("  %+v \n", " Запуск сервиса Авторизации")
	сервер := &СхемаСервера{
		Имя:   "SynQuic",
		Адрес: "localhost:4242",
		ДанныеСессии: ДанныеСессии{
			Блок:   &sync.RWMutex{},
			Потоки: []quic.Stream{},
		},
	}
	сообщениеРегистрации := Сообщение{
		Сервис:      Сервис,
		Регистрация: true,
		Маршруты:    []Маршрут{"reg", "auth", "verify", "checkLogin", "code"},
	}

	клиент.Соединиться(сервер,
		сообщениеРегистрации,
		ОбработчикОтветаРегистрации,
		ОбработчикЗапросовСервера)
}

func ИнициализацияБазыДанных() {
	БазаДанных = Таблица{
		Имя:           "клиенты",
		Дирректория:   "./database",
		ПервичныйКлюч: "Email", //
		База:          База{},
		// Индексы: ТаблицаИндексов{
		// 	Версия:      1,
		// 	Дирректория: "./database/index",
		// 	ИмяТаблицы:  "клиенты",
		// 	// КартаИндексов: map[ПутьИндекса]struct{}{
		// 	// 	"Адрес.Город": struct{}{},
		// 	// },
		// },
		УникальныеИндексы: ТаблицаИндексов{
			Версия:      1,
			Дирректория: "./database/uniq_index",
			ИмяТаблицы:  "клиенты",
			КартаИндексов: map[ПутьИндекса]struct{}{
				"Email": {},
				"Логин": {},
			},
			Уникальный: true,
		},
	}

	ошибкаБазы := БазаДанных.АктивироватьТаблицу()
	if ошибкаБазы.Код != 0 {
		Ошибка(" ошибкаБазы %+v \n", ошибкаБазы)
	}
}

func ОбработчикОтветаРегистрации(сообщение Сообщение) {
	Инфо("  ОбработчикОтветаРегистрации %+v \n", сообщение)
}

func ОбработчикЗапросовСервера(поток quic.Stream, сообщение Сообщение) {
	Инфо("  ОбработчикЗапросовСервера %+v \n", сообщение)
	var err error
	var ok bool
	var email string
	параметрыЗапроса, err := url.Parse(сообщение.Запрос.МаршрутЗапроса)
	Инфо(" параметрыЗапроса %+v \n", параметрыЗапроса)
	if err != nil {
		Ошибка("Ошибка при парсинге СтрокаЗапроса запроса:", err)
	}

	параметрыЗапроса.Path = strings.Trim(параметрыЗапроса.Path, "/")
	дейсвтия := strings.Split(параметрыЗапроса.Path, "/")

	var Действие string
	if len(дейсвтия) == 0 {
		Инфо(" Пустой маршрут, добавляем в маршруты обработку по умолчанию.... \n")
		// Читаем заголовки парсим и проверяем JWT
		Действие = "verify"

	} else {
		Действие = дейсвтия[0]
	}

	switch Действие {
	case "reg":
		ok, err = Регистрация(&сообщение)
		Инфо(" Регистрация ок  %+v \n", ok)
	case "auth":
		ok, email, err = Авторизация(&сообщение)
		if ok && email != "" {
			ОтправитьПроверочныйКод(&сообщение, email)
		}
		Инфо(" Авторизация ок  %+v \n", ok)
	case "code":
		ok, err = ПроверитьКод(&сообщение)
		Инфо(" ПроверитьКод ок  %+v \n", ok)
	case "checkLogin":
		ok, err = ЛогинСвободен(&сообщение)
		Инфо(" ПроверитьЛогин ок  %+v \n", ok)
	case "verify":
		ok, err = ВлаидацияТокена(&сообщение)
		Инфо(" ВлаидацияТокена ок  %+v \n", ok)
	default:
		// если не совпадает ни с одним из действий то проверим если токен не пустой то проверим подпись
		ok, err = ВлаидацияТокена(&сообщение)
		Инфо(" ВлаидацияТокена ок  %+v \n", ok)
	}

	if err != nil {
		Ошибка(" Генерируем сообщение ощибки или возвращаем сообщение ошибки %+v \n", err)
		ответ := сообщение.Ответ[Сервис]
		ответ.Сервис = Сервис
		ответ.ТипОтвета = Error
		if ответ.Данные != nil {
			// если в данных уже есть какаято инфа, заполенная одной из функций , то добавим ошбку
			ответ.Данные.(map[string]string)["error"] = err.Error()
		} else {
			ответ.Данные = err
		}

		ответ.ЗапросОбработан = true
		сообщение.Ответ[Сервис] = ответ
	}

	отправить, err := Кодировать(сообщение)
	if err != nil {
		Ошибка("  %+v \n", err)
	}
	поток.Write(отправить)

}

func Авторизация(сообщение *Сообщение) (bool, string, error) {
	форма := сообщение.Запрос.Форма
	if len(форма) > 0 {
		var логин, пароль []string

		if логин = форма["login"]; len(логин) > 0 && логин[0] == "" {
			return false, "", errors.New("нет логина")
		}
		if пароль = форма["password"]; len(пароль) > 0 && пароль[0] == "" {
			return false, "", errors.New("нет пароля")
		}

		status, токен, email, err := ПроверитьДанныеВБД(сообщение.ИдКлиента.String(), логин[0], пароль[0])
		if err != nil {
			return status, "", err
		}
		сообщение.ТокенКлиента = токен
		JWT, err := СоздатьJWT(токен)
		if err != nil {
			Ошибка(" не удалось создать токен  %+v \n", err)
			return false, "", err
		}

		сообщение.JWT = JWT

		return status, email, nil
	} else {
		return false, "", errors.New("нет данных для авторизации, логин и пароль обязательны")
	}
}

func ОтправитьПроверочныйКод(сообщение *Сообщение, email string) {
	Инфо(" ОтправитьПроверочныйКод ? отправялем сообщение на почту или смс %+v \n")

	// в сообщение в ответ добавляем токен для сверки с кодом, который будет подставлен в форму подтверждения
	ответ := сообщение.Ответ[Сервис]
	токенВерификации := СоздатьТокенОбновления(16)
	ответ.Данные = map[string]string{
		"токенВерификации": токенВерификации,
	}
	ответ.Сервис = Сервис
	ответ.ЗапросОбработан = true

	проверочныйКод := СгенерироватьПроверочныйКод(6)
	СохранитьКодАвторизации(email, сообщение.ИдКлиента.String(), проверочныйКод, токенВерификации)

	сообщение.Ответ[Сервис] = ответ

	go ОтправитьEmail([]string{email}, "Проверочный код", "Введите проверочный код на странице атворизации "+проверочныйКод)

}

func ЛогинСвободен(логин string) (bool, error) {
	Инфо("ПроверитьЛогин , нужно проверить логин на свободу \n")
	данные, ошибка := БазаДанных.Найти("Логин", логин)
	if ошибка.Код == ОшибкаКлючНеНайден {
		Инфо("  %+v  %+v \n", данные, ошибка)
		return true, nil
	} else if ошибка.Код == Ок {
		Инфо("  %+v  %+v \n", данные, ошибка)
		if len(данные) > 0 {
			return false, errors.New("логин занят")
		}
		return false, errors.New("логин занят")
	}
	Инфо("  %+v  %+v \n", данные, ошибка)
	return true, nil
}
func EmailСвободен(email string) (bool, error) {
	Инфо("ПроверитьEmail , проверить email на сущестование \n")
	Инфо("ПроверитьЛогин , нужно проверить логин на свободу \n")
	данные, ошибка := БазаДанных.Найти("Email", email)
	if ошибка.Код == ОшибкаКлючНеНайден {
		Инфо("  %+v  %+v \n", данные, ошибка)
		return true, nil
	} else if ошибка.Код == Ок {
		Инфо("  %+v  %+v \n", данные, ошибка)
		if len(данные) > 0 {
			return false, errors.New("логин занят")
		}
		return false, errors.New("логин занят")
	}
	Инфо("  %+v  %+v \n", данные, ошибка)
	return true, nil
}

func ПроверитьКод(сообщение *Сообщение) (bool, error) {
	Инфо("ПроверитьКод , нужно проверить отправленный смс код или код на email \n")
	файл, err := os.ReadFile("authCode/" + сообщение.ИдКлиента.String())
	if err != nil {
		Ошибка("  %+v \n", err)
	}
	// кодПроверки := map[string]string{
	// 	токенВерификации: проверочныйКод,
	// 	"email":          email,
	// }
	кодПроверки := make(map[string]string)
	json.Unmarshal(файл, &кодПроверки)

	форма := сообщение.Запрос.Форма
	токенВерификации, естьТокен := форма["token"]
	if !естьТокен || len(токенВерификации) < 1 {
		Ошибка(" Нет токена верификации в данных формы %+v \n", форма)
	}
	код, естьКод := форма["code"]
	if !естьКод || len(код) < 1 {
		Ошибка(" Нет кода подтверждения в данных формы %+v \n", форма)
	}

	if СохранённыйКод, есть := кодПроверки[токенВерификации[0]]; есть {
		if СохранённыйКод == код[0] {

			ответ := сообщение.Ответ[Сервис]
			ответ.Сервис = Сервис
			ответ.ЗапросОбработан = true
			ответ.Данные = map[string]string{
				"статусПроверкиКода": "Проверочный код верный",
			}

			return true, nil
		} else {
			if кодПроверки["email"] != "" {
				//
				ОтправитьПроверочныйКод(сообщение, кодПроверки["email"])
			}

		}
	}

	return true, nil
}

func Регистрация(сообщение *Сообщение) (bool, error) {
	ответ := сообщение.Ответ[Сервис]

	ответ.Сервис = Сервис
	ответ.ЗапросОбработан = true

	форма := сообщение.Запрос.Форма
	if len(форма) > 0 {
		var логин, пароль, email []string
		if логин = форма["login"]; len(логин) > 0 && логин[0] == "" {
			ответ.Данные = map[string]string{
				"СтатусРегистрации": "Нет логина",
			}
			return false, errors.New("Нет логина")
		}
		if пароль = форма["password"]; len(пароль) > 0 && пароль[0] == "" {
			ответ.Данные = map[string]string{
				"СтатусРегистрации": "Нет пароля",
			}
			return false, errors.New("Нет пароля")
		}

		свободен, err := ЛогинСвободен(логин[0])
		if !свободен && err != nil {
			ответ.Данные = map[string]string{
				"СтатусРегистрации": "Логин занят",
			}
			return false, err
		}
		if email = форма["email"]; len(email) > 0 && email[0] == "" {
			ответ.Данные = map[string]string{
				"СтатусРегистрации": "Нет email",
			}

			return false, errors.New("Нет email")
		}
		emailСвободен, err := EmailСвободен(email[0])
		if !emailСвободен && err != nil {
			ответ.Данные = map[string]string{
				"СтатусРегистрации": "email уже заригистрирован",
			}
			return false, err
		}
	}

	новыйТокенКлиент := ТокенКлинета{
		ИдКлиента: сообщение.ИдКлиента,
		Роль:      []string{"клиент"},
		Токен:     СоздатьТокенОбновления(16),
		Права:     []string{"клиент"},
		Истекает:  time.Now().Add(60 * time.Minute).Unix(),
		Создан:    time.Now().Unix(),
	}

	JWT, err := СоздатьJWT(новыйТокенКлиент)
	if err != nil {
		Ошибка(" не удалось создать токен  %+v \n", err)
		return false, err
	}
	сообщение.JWT = JWT
	err = СохранитьКлиентаВБД(сообщение, новыйТокенКлиент)
	if err != nil {
		Ошибка(" не удалось сохранить в БД  %+v \n", err)
		return false, err
	}

	ответ.Данные = map[string]string{
		"СтатусРегистрации": "успех",
	}
	сообщение.Ответ[Сервис] = ответ
	return true, nil
}

func ПроверитьДанныеВБД(ИдКлиента string, логин string, пароль string) (bool, ТокенКлинета, string, error) {
	файл, err := os.ReadFile("users/" + ИдКлиента)
	if err != nil {
		Ошибка("  %+v \n", err)
	}
	var данныеКлиента ДанныеКлиента
	json.Unmarshal(файл, &данныеКлиента)

	клиент := данныеКлиента.Данные
	if клиент["login"][0] != логин {
		Ошибка("  %+v \n", err)
		return false, ТокенКлинета{}, "", errors.New("Неверный логин")
	}
	if клиент["password"][0] != пароль {
		Ошибка("  %+v \n", err)
		return false, ТокенКлинета{}, "", errors.New("Неверный пароль")
	}

	токен := данныеКлиента.JWT
	токен.Токен = СоздатьТокенОбновления(16)
	токен.Истекает = time.Now().Add(60 * time.Minute).Unix()
	токен.Создан = time.Now().Unix()

	if клиент["email"][0] == "" {
		Ошибка("  %+v \n", err)
		return false, ТокенКлинета{}, "", errors.New("Не задан email")
	}

	return true, токен, клиент["email"][0], nil

}

type ДанныеКлиента struct {
	Данные map[string][]string
	JWT    ТокенКлинета
}

func СохранитьКлиентаВБД(сообщение *Сообщение, новыйТокенКлиент ТокенКлинета) error {

	форма := сообщение.Запрос.Форма
	клиент := ДанныеКлиента{
		Данные: форма,
		JWT:    новыйТокенКлиент,
	}

	данныеДляСохранения, err := json.Marshal(клиент)
	if err != nil {
		Ошибка("  %+v \n", err)
	}
	err = os.WriteFile("users/"+сообщение.ИдКлиента.String(), данныеДляСохранения, 0644)
	if err != nil {
		Ошибка("  %+v \n", err)
	}

	// TODO: Сохранить клиента в БД
	Инфо(" Сохранить клиента в БД  \n")
	return nil
}

func СоздатьJWT(данныеТокена ТокенКлинета) (string, error) {

	claims := jwt.MapClaims{
		"UID":     данныеТокена.ИдКлиента,
		"role":    данныеТокена.Роль,
		"token":   данныеТокена.Токен,
		"access":  данныеТокена.Права,
		"expires": данныеТокена.Истекает,
		"created": данныеТокена.Создан,
	}
	return ПодписатьJWT(claims, СоздатьСекретКлиента(данныеТокена.ИдКлиента))
}

func ПодписатьJWT(данныеJWT jwt.MapClaims, секрет string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, данныеJWT)

	// Подписываем токен с использованием секретного ключа
	подписаннаяСтрока, err := token.SignedString([]byte(секрет))
	if err != nil {
		return "", err
	}

	return подписаннаяСтрока, nil
}

func ПолучитьСекретныйКлючКлиента(ИдКлиента uuid.UUID) string {
	// откроем файл в котором храниться скертный ключ
	файл, err := os.Open("secrets/" + ИдКлиента.String())

	if err != nil {
		Ошибка("  %+v \n", err)
		return ""
	}
	defer файл.Close()
	секретныйКлюч := make([]byte, 256)
	// прочитаем содержимое файла и вернём
	_, err = файл.Read(секретныйКлюч)
	if err != nil {
		Ошибка("  %+v \n", err)
	}

	return string(секретныйКлюч)
}

func СоздатьТокенОбновления(размер int) string {
	key := make([]byte, размер)
	_, err := rand.Read(key)
	if err != nil {
		return ""
	}

	// Кодируем байты в base64 строку
	keyString := base64.URLEncoding.EncodeToString(key)
	return keyString
}
func СоздатьСекретКлиента(ИдКлиента uuid.UUID) string {
	// Генерируем байты случайных данных
	key := make([]byte, 256)
	_, err := rand.Read(key)
	if err != nil {
		return ""
	}

	// Кодируем байты в base64 строку
	keyString := base64.URLEncoding.EncodeToString(key)

	// запишем ключ в файл
	err = os.WriteFile("secrets/"+ИдКлиента.String(), []byte(keyString), 0644)
	if err != nil {
		Ошибка("  %+v \n", err)
	}
	return keyString
}
func ВлаидацияТокена(сообщение *Сообщение) (bool, error) {
	секрет := ПолучитьСекретныйКлючКлиента(сообщение.ИдКлиента)
	if секрет == "" {

		return false, errors.New("не удалось получить секретный ключ клиента")
	}
	token, err := jwt.Parse(сообщение.JWT, func(token *jwt.Token) (interface{}, error) {
		return []byte(секрет), nil
	})
	if err != nil {

		return false, err
	}

	// Проверяем валидность токена
	if токен, ok := token.Claims.(jwt.MapClaims); !ok || !token.Valid {
		Ошибка(" токен не валидный %+v \n", токен)
		сообщение.JWT = "invaild"
		return false, nil
	} else {
		истекает := time.Unix(токен["expires"].(int64), 0)

		// если осталось менее 5 минут переподпишем токен

		if осталосьВремениДоИстечения := time.Now().Sub(истекает); time.Duration(осталосьВремениДоИстечения.Minutes()) < 5*time.Minute {

			токен["token"] = СоздатьТокенОбновления(16)
			токен["expires"] = time.Now().Add(60 * time.Minute).Unix()
			токен["created"] = time.Now().Unix()

			новыйСекрет := СоздатьСекретКлиента(токен["UID"].(uuid.UUID))
			новыйJWT, err := ПодписатьJWT(токен, новыйСекрет)
			if err != nil {
				Ошибка("  %+v \n", err)
			}
			сообщение.JWT = новыйJWT

			ответ := сообщение.Ответ[Сервис]

			ответ.Сервис = Сервис
			ответ.ЗапросОбработан = true
			ответ.Данные = map[string]bool{
				"ТокенВерный": true,
			}

		}
	}

	return true, nil
}

func СохранитьКодАвторизации(email string, ИдКлиента string, проверочныйКод string, токенВерификации string) error {

	кодПроверки := map[string]string{
		токенВерификации: проверочныйКод,
		"email":          email,
	}

	данныеДляСохранения, err := json.Marshal(кодПроверки)
	if err != nil {
		Ошибка("  %+v \n", err)
		return err
	}
	err = os.WriteFile("authCode/"+ИдКлиента, данныеДляСохранения, 0644)
	if err != nil {
		Ошибка("  %+v \n", err)
		return err
	}
	return nil
}
func ОтправитьEmail(кому []string, тема string, тело string) error {
	from := "79880970078@ya.ru"
	password := "Satori@27$"
	smtpHost := "smtp.yandex.ru"
	smtpPort := "465"

	auth := smtp.PlainAuth("", from, password, smtpHost)

	msg := []byte("To: " + кому[0] + "\r\n" +
		"Subject: " + тема + "\r\n" +
		"\r\n" +
		тело + "\r\n")

	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, кому, msg)
	if err != nil {
		return err
	}

	return nil
}
func СгенерироватьПроверочныйКод(КоличествоСимволов int) string {
	код, err := rand.Int(
		rand.Reader,
		big.NewInt(int64(math.Pow(10, float64(КоличествоСимволов)))),
	)
	if err != nil {
		panic(err)
	}

	str := код.String()
	if len(str) < КоличествоСимволов {
		str = "0" + str
	}
	return str
}
