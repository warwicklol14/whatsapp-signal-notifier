project_name=whatsapp-signal-notifier

mkdir build
cd build || exit

mkdir -p $project_name/appdata
cp ../appdata/switch_to_signal_video.mp4 $project_name/appdata/

file_name=$project_name/$project_name.exe
GOOS=windows go build -o $file_name ../
zip -r $project_name-windows $project_name/
rm $file_name

file_name=$project_name/$project_name
GOOS=darwin go build -o $file_name ../
zip -r $project_name-mac $project_name/
rm $file_name

file_name=$project_name/$project_name
GOOS=linux go build -o $file_name ../
zip -r $project_name-linux $project_name/
rm $file_name

rm -r $project_name
